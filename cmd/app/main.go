// Package main provides the entry point for the application with CLI commands.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
)

// Build-time version information (injected via ldflags during build).
var (
	version   = "v0.22.0" // Semantic version with "v" prefix (e.g., "v0.12.0")
	buildDate = "unknown" // ISO 8601 build timestamp
	commitSHA = "unknown" // Git commit SHA
)

func main() {
	// Custom version printer to display build metadata
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("Version:    %s\n", version)
		fmt.Printf("Build Date: %s\n", buildDate)
		fmt.Printf("Commit SHA: %s\n", commitSHA)
	}

	cmd := &cli.Command{
		Name:     "secrets",
		Usage:    "A lightweight secrets manager designed for simplicity and security",
		Version:  version,
		Commands: getCommands(version),
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}
