package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/fx"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/config"
	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/logger"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/scheduler"
	"crossshare-server/internal/server"
	"crossshare-server/internal/service"
	"crossshare-server/internal/settings"
	"crossshare-server/internal/storage"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "dump-config":
			runDumpConfig(os.Args[2:])
			return
		case "install":
			runInstall(os.Args[2:])
			return
		case "--version", "-v", "version":
			fmt.Println(version)
			return
		}
	}

	fx.New(
		fx.Supply(server.Version(version)),
		config.Module,
		logger.Module,
		crypto.Module,
		db.Module,
		settings.Module,
		audit.Module,
		migration.Module,
		storage.Module,
		service.Module,
		scheduler.Module,
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
