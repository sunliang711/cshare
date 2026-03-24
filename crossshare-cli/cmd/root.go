package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	serverURL string
	token     string
	noProxy   bool
	verbose   bool
	version   string
)

var debugLog = log.New(io.Discard, "[debug] ", 0)

var rootCmd = &cobra.Command{
	Use:   "share",
	Short: "CrossShare CLI - share text and files across devices",
	Long: `CrossShare CLI client for crossshare-server.

Push text or files, pull them by key, and manage shared content
from the command line.

Environment variables:
  CROSSSHARE_SERVER   Server base URL (default: http://localhost:10431)
  CROSSSHARE_TOKEN    JWT auth token

Flags --server and --token take precedence over environment variables.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			debugLog.SetOutput(os.Stderr)
			debugLog.Printf("server: %s", serverURL)
			if token != "" {
				debugLog.Printf("auth: token set")
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Printf("share version %s\n", version)
			return
		}
		cmd.Help()
	},
}

func SetVersion(v string) {
	version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Bool("version", false, "show version")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output to stderr")
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", envOrDefault("CROSSSHARE_SERVER", "http://localhost:10431"), "server base URL")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", os.Getenv("CROSSSHARE_TOKEN"), "JWT auth token")
	rootCmd.PersistentFlags().BoolVar(&noProxy, "noproxy", false, "ignore HTTP_PROXY/HTTPS_PROXY environment variables")
}

func newClient() *Client {
	return NewClient(serverURL, token, noProxy)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
