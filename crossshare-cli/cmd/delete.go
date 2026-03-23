package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete shared content by key",
	Long: `Delete content from crossshare-server using a share key.

Examples:
  share delete A8k2dP`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		c := newClient()

		resp, err := c.doJSON("DELETE", "/pull/"+key, nil, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if resp.Code != 0 {
			fmt.Fprintf(os.Stderr, "Delete failed: [%d] %s\n", resp.Code, resp.Msg)
			os.Exit(1)
		}

		var data struct {
			Key     string `json:"key"`
			Deleted bool   `json:"deleted"`
		}
		json.Unmarshal(resp.Data, &data)

		fmt.Printf("Deleted: %s\n", data.Key)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
