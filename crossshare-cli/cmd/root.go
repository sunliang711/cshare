package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	serverURL string
	token     string
	noProxy   bool
)

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
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
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
