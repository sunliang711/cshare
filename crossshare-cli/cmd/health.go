package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health status",
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		resp, err := c.doJSON("GET", "/health", nil, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if resp.Code != 0 {
			fmt.Fprintf(os.Stderr, "Server unhealthy: %s\n", resp.Msg)
			os.Exit(1)
		}

		var data struct {
			Service string `json:"service"`
			Status  string `json:"status"`
			Time    string `json:"time"`
		}
		json.Unmarshal(resp.Data, &data)

		fmt.Printf("Service:  %s\n", data.Service)
		fmt.Printf("Status:   %s\n", data.Status)
		fmt.Printf("Time:     %s\n", data.Time)
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
