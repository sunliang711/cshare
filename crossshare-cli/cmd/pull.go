package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	pullOutput      string
	pullJSON        bool
	pullDeleteAfter bool
)

var pullCmd = &cobra.Command{
	Use:   "pull <key>",
	Short: "Pull shared content by key",
	Long: `Retrieve content from crossshare-server using a share key.

Examples:
  share pull A8k2dP                  # auto-detect: print text or save binary
  share pull A8k2dP -o output.bin    # save to specific file
  share pull A8k2dP --json           # force JSON response (text only)
  share pull A8k2dP --delete         # delete after pulling`,
	Args: cobra.ExactArgs(1),
	Run:  runPull,
}

func init() {
	pullCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "save content to file")
	pullCmd.Flags().BoolVar(&pullJSON, "json", false, "force JSON response (text content)")
	pullCmd.Flags().BoolVar(&pullDeleteAfter, "delete", false, "delete content after pulling")
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) {
	key := args[0]
	c := newClient()

	if pullJSON {
		pullAsJSON(c, key)
		return
	}

	pullAsStream(c, key)
}

func pullAsJSON(c *Client, key string) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	if pullDeleteAfter {
		headers["Delete-After-Pull"] = "true"
	}

	resp, err := c.doJSON("GET", "/pull/"+key, nil, headers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Code != 0 {
		fmt.Fprintf(os.Stderr, "Pull failed: [%d] %s\n", resp.Code, resp.Msg)
		os.Exit(1)
	}

	var data struct {
		Key         string `json:"key"`
		Text        string `json:"text"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int    `json:"size"`
		Deleted     bool   `json:"deleted"`
	}
	json.Unmarshal(resp.Data, &data)

	if pullOutput != "" {
		if err := os.WriteFile(pullOutput, []byte(data.Text), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Saved to %s (%s)\n", pullOutput, humanSize(int64(data.Size)))
	} else {
		fmt.Print(data.Text)
	}

	if data.Deleted {
		fmt.Fprintln(os.Stderr, "(key deleted after pull)")
	}
}

func pullAsStream(c *Client, key string) {
	headers := map[string]string{}
	if pullDeleteAfter {
		headers["Delete-After-Pull"] = "true"
	}

	resp, err := c.doRaw("GET", "/pull/"+key, nil, headers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var apiResp APIResponse
		if json.Unmarshal(body, &apiResp) == nil && apiResp.Code != 0 {
			fmt.Fprintf(os.Stderr, "Pull failed: [%d] %s\n", apiResp.Code, apiResp.Msg)
		} else {
			fmt.Fprintf(os.Stderr, "Pull failed: HTTP %d\n", resp.StatusCode)
		}
		os.Exit(1)
	}

	shareType := resp.Header.Get("Crossshare-Type")
	filename := resp.Header.Get("Crossshare-Filename")
	deleted := resp.Header.Get("Key-Deleted") == "true"

	output := pullOutput
	if output == "" && shareType == "File" && filename != "" {
		output = filename
	}

	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
			os.Exit(1)
		}
		n, err := io.Copy(f, resp.Body)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Saved to %s (%s)\n", output, humanSize(n))
	} else {
		io.Copy(os.Stdout, resp.Body)
	}

	if deleted {
		fmt.Fprintln(os.Stderr, "(key deleted after pull)")
	}
}
