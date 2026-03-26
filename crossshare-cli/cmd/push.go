package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	pushFile        string
	pushTTL         int
	pushFilename    string
	pushContentType string
)

var pushCmd = &cobra.Command{
	Use:   "push [text]",
	Short: "Push text or a file to the server",
	Long: `Push content to crossshare-server and receive a key for retrieval.

Examples:
  share push "hello world"                   # push text
  share push "hello" --ttl 7200              # push text with 2h TTL
  share push                                 # read text from stdin
  echo "piped" | share push                  # pipe text from stdin
  share push -f ./report.pdf                 # push a file
  share push -f ./notes.txt --filename a.txt # push file with custom name`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPush,
}

func init() {
	pushCmd.Flags().StringVarP(&pushFile, "file", "f", "", "file path to upload")
	pushCmd.Flags().IntVar(&pushTTL, "ttl", 0, "TTL in seconds (default: server default)")
	pushCmd.Flags().StringVar(&pushFilename, "filename", "", "custom filename")
	pushCmd.Flags().StringVar(&pushContentType, "content-type", "", "custom content type")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) {
	if pushFile != "" {
		pushBinaryFile()
		return
	}

	var text string
	if len(args) == 1 && args[0] != "-" {
		text = args[0]
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		text = string(data)
	}

	pushTextContent(text)
}

func pushTextContent(text string) {
	c := newClient()

	body := map[string]any{
		"text": text,
	}
	if pushTTL > 0 {
		body["ttl"] = pushTTL
	}
	if pushFilename != "" {
		body["filename"] = pushFilename
	}
	if pushContentType != "" {
		body["content_type"] = pushContentType
	}

	resp, err := c.doJSON("POST", "/push/text", body, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Code != 0 {
		fmt.Fprintf(os.Stderr, "Push failed: [%d] %s\n", resp.Code, resp.Msg)
		os.Exit(1)
	}

	printPushResult(resp.Data)
}

func pushBinaryFile() {
	data, filename, err := readFileData(pushFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if pushFilename != "" {
		filename = pushFilename
	}

	c := newClient()
	headers := map[string]string{
		"Content-Type": "application/octet-stream",
	}
	if filename != "" {
		headers["Filename"] = filename
	}
	if pushTTL > 0 {
		headers["X-TTL"] = strconv.Itoa(pushTTL)
	}
	if pushContentType != "" {
		headers["X-Content-Type"] = pushContentType
	}

	resp, err := c.doRaw("POST", "/push/binary", bytes.NewReader(data), headers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error: unexpected server response\n")
		os.Exit(1)
	}

	if apiResp.Code != 0 {
		fmt.Fprintf(os.Stderr, "Push failed: [%d] %s\n", apiResp.Code, apiResp.Msg)
		os.Exit(1)
	}

	printPushResult(apiResp.Data)
}

func readFileData(path string) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		return data, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file %s: %w", path, err)
	}
	return data, filepath.Base(path), nil
}

func printPushResult(data json.RawMessage) {
	var result struct {
		Key      string `json:"key"`
		TTL      int    `json:"ttl"`
		Size     int    `json:"size"`
		Type     string `json:"type"`
		Filename string `json:"filename"`
		ExpireAt int64  `json:"expire_at"`
	}
	json.Unmarshal(data, &result)

	fmt.Printf("Key:      %s\n", result.Key)
	fmt.Printf("Type:     %s\n", result.Type)
	fmt.Printf("Size:     %s\n", humanSize(int64(result.Size)))
	fmt.Printf("TTL:      %s\n", humanDuration(result.TTL))
	if result.Filename != "" {
		fmt.Printf("Filename: %s\n", result.Filename)
	}
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMG"[exp])
}

func humanDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}
