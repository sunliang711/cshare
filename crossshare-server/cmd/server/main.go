package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/fx"

	"crossshare-server/internal/config"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/logger"
	"crossshare-server/internal/server"
	"crossshare-server/internal/service"
	"crossshare-server/internal/storage"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "dump-config":
			runDumpConfig(os.Args[2:])
			return
		}
	}

	fx.New(
		config.Module,
		logger.Module,
		storage.Module,
		service.Module,
		handler.Module,
		server.Module,
	).Run()
}

func runDumpConfig(args []string) {
	fs := flag.NewFlagSet("dump-config", flag.ExitOnError)
	output := fs.String("o", "", "output file path (default: stdout)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: crossshare-server dump-config [-o <file>]\n\nDump the default configuration in YAML format.\n\nOptions:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	data, err := config.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		os.Stdout.Write(data)
		return
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *output, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Default config written to %s\n", *output)
}
