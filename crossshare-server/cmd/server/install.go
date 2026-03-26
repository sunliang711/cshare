package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"crossshare-server/internal/config"
)

const (
	defaultInstallDir  = "/usr/local/bin"
	defaultConfigDir   = "/etc/crossshare"
	defaultServiceName = "crossshare-server"
	systemdDir         = "/etc/systemd/system"
)

var serviceTemplate = `[Unit]
Description=CrossShare Server
Documentation=https://github.com/sunliang711/cshare
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	binDir := fs.String("bin-dir", defaultInstallDir, "directory to install the binary")
	confDir := fs.String("config-dir", defaultConfigDir, "directory for config files")
	configFile := fs.String("config", "", "path to config file to install (optional, generates default if omitted)")
	serviceName := fs.String("name", defaultServiceName, "systemd service name")
	user := fs.String("user", "root", "user to run the service as")
	group := fs.String("group", "root", "group to run the service as")
	noEnable := fs.Bool("no-enable", false, "do not enable the service after install")
	start := fs.Bool("start", false, "start the service after install")
	uninstall := fs.Bool("uninstall", false, "uninstall the service instead of installing")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: crossshare-server install [options]

Install crossshare-server as a systemd service on Linux.

This command will:
  1. Copy the binary to the install directory
  2. Copy or generate a config file
  3. Create a systemd service unit file
  4. Reload systemd and enable the service

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  sudo crossshare-server install
  sudo crossshare-server install --config /path/to/config.yaml
  sudo crossshare-server install --user crossshare --group crossshare
  sudo crossshare-server install --start
  sudo crossshare-server install --uninstall
`)
	}
	fs.Parse(args)

	if runtime.GOOS != "linux" {
		fatalf("install command is only supported on Linux (current OS: %s)", runtime.GOOS)
	}

	if os.Geteuid() != 0 {
		fatalf("install command must be run as root (try: sudo crossshare-server install)")
	}

	if *uninstall {
		doUninstall(*binDir, *confDir, *serviceName)
		return
	}

	doInstall(*binDir, *confDir, *configFile, *serviceName, *user, *group, *noEnable, *start)
}

func doInstall(binDir, confDir, configFile, serviceName, user, group string, noEnable, start bool) {
	fmt.Println("==> Installing crossshare-server as a systemd service...")

	// 1. Copy binary
	selfPath, err := os.Executable()
	if err != nil {
		fatalf("failed to determine executable path: %v", err)
	}
	selfPath, err = filepath.EvalSymlinks(selfPath)
	if err != nil {
		fatalf("failed to resolve executable path: %v", err)
	}

	destBin := filepath.Join(binDir, defaultServiceName)
	if selfPath != destBin {
		fmt.Printf("  - Copying binary to %s\n", destBin)
		if err := os.MkdirAll(binDir, 0755); err != nil {
			fatalf("failed to create directory %s: %v", binDir, err)
		}
		if err := copyFile(selfPath, destBin, 0755); err != nil {
			fatalf("failed to copy binary: %v", err)
		}
	} else {
		fmt.Printf("  - Binary already at %s, skipping copy\n", destBin)
	}

	// 2. Config directory and file
	fmt.Printf("  - Setting up config directory %s\n", confDir)
	if err := os.MkdirAll(confDir, 0755); err != nil {
		fatalf("failed to create config directory %s: %v", confDir, err)
	}

	destConf := filepath.Join(confDir, "config.yaml")
	if configFile != "" {
		// Copy user-supplied config
		fmt.Printf("  - Copying config file to %s\n", destConf)
		if err := copyFile(configFile, destConf, 0644); err != nil {
			fatalf("failed to copy config file: %v", err)
		}
	} else if _, err := os.Stat(destConf); os.IsNotExist(err) {
		// Generate default config
		fmt.Printf("  - Generating default config at %s\n", destConf)
		data, err := defaultConfigBytes()
		if err != nil {
			fatalf("failed to generate default config: %v", err)
		}
		if err := os.WriteFile(destConf, data, 0644); err != nil {
			fatalf("failed to write config file: %v", err)
		}
	} else {
		fmt.Printf("  - Config file already exists at %s, skipping\n", destConf)
	}

	// 3. Create systemd service file
	serviceFile := filepath.Join(systemdDir, serviceName+".service")
	fmt.Printf("  - Writing systemd unit file to %s\n", serviceFile)

	content := fmt.Sprintf(serviceTemplate, destBin, confDir, confDir)

	// Add User/Group if not root
	if user != "root" || group != "root" {
		lines := strings.Split(content, "\n")
		var result []string
		for _, line := range lines {
			result = append(result, line)
			if strings.HasPrefix(line, "Type=") {
				if user != "root" {
					result = append(result, fmt.Sprintf("User=%s", user))
				}
				if group != "root" {
					result = append(result, fmt.Sprintf("Group=%s", group))
				}
			}
		}
		content = strings.Join(result, "\n")
	}

	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		fatalf("failed to write service file: %v", err)
	}

	// 4. Reload systemd
	fmt.Println("  - Reloading systemd daemon...")
	if err := runCmd("systemctl", "daemon-reload"); err != nil {
		fatalf("failed to reload systemd: %v", err)
	}

	// 5. Enable the service
	if !noEnable {
		fmt.Printf("  - Enabling service %s...\n", serviceName)
		if err := runCmd("systemctl", "enable", serviceName); err != nil {
			fatalf("failed to enable service: %v", err)
		}
	}

	// 6. Optionally start the service
	if start {
		fmt.Printf("  - Starting service %s...\n", serviceName)
		if err := runCmd("systemctl", "start", serviceName); err != nil {
			fatalf("failed to start service: %v", err)
		}
	}

	fmt.Println()
	fmt.Println("==> Installation complete!")
	fmt.Println()
	fmt.Printf("  Binary:   %s\n", destBin)
	fmt.Printf("  Config:   %s\n", destConf)
	fmt.Printf("  Service:  %s\n", serviceFile)
	fmt.Println()
	if !start {
		fmt.Printf("  Start with: sudo systemctl start %s\n", serviceName)
	}
	fmt.Printf("  View logs: sudo journalctl -u %s -f\n", serviceName)
	fmt.Printf("  Status:    sudo systemctl status %s\n", serviceName)
}

func doUninstall(binDir, confDir, serviceName string) {
	fmt.Println("==> Uninstalling crossshare-server systemd service...")

	serviceFile := filepath.Join(systemdDir, serviceName+".service")

	// 1. Stop the service (ignore errors — it may not be running)
	fmt.Printf("  - Stopping service %s...\n", serviceName)
	_ = runCmd("systemctl", "stop", serviceName)

	// 2. Disable the service
	fmt.Printf("  - Disabling service %s...\n", serviceName)
	_ = runCmd("systemctl", "disable", serviceName)

	// 3. Remove service file
	if _, err := os.Stat(serviceFile); err == nil {
		fmt.Printf("  - Removing %s\n", serviceFile)
		if err := os.Remove(serviceFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", serviceFile, err)
		}
	}

	// 4. Reload systemd
	fmt.Println("  - Reloading systemd daemon...")
	_ = runCmd("systemctl", "daemon-reload")

	// 5. Remove binary
	destBin := filepath.Join(binDir, defaultServiceName)
	if _, err := os.Stat(destBin); err == nil {
		fmt.Printf("  - Removing binary %s\n", destBin)
		if err := os.Remove(destBin); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", destBin, err)
		}
	}

	fmt.Println()
	fmt.Println("==> Uninstall complete!")
	fmt.Printf("  Note: config directory %s was NOT removed. Remove it manually if desired:\n", confDir)
	fmt.Printf("    sudo rm -rf %s\n", confDir)
}

// --- Helpers ---

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("create destination %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}
	return out.Close()
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultConfigBytes() ([]byte, error) {
	return config.DefaultConfig()
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
