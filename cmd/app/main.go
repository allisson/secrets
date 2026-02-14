// Package main provides the entry point for the application with CLI commands.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/allisson/secrets/cmd/app/commands"
)

func main() {
	cmd := &cli.Command{
		Name:    "app",
		Usage:   "Go project template application",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start the HTTP server",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunServer(ctx)
				},
			},
			{
				Name:  "migrate",
				Usage: "Run database migrations",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunMigrations()
				},
			},
			{
				Name:  "create-master-key",
				Usage: "Generate a new Master Key for envelope encryption",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "id",
						Aliases: []string{"i"},
						Value:   "",
						Usage:   "Master key ID (e.g., prod-master-key-2025)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCreateMasterKey(cmd.String("id"))
				},
			},
			{
				Name:  "create-kek",
				Usage: "Create a new Key Encryption Key (KEK)",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "algorithm",
						Aliases: []string{"alg"},
						Value:   "aes-gcm",
						Usage:   "Encryption algorithm to use (aes-gcm or chacha20-poly1305)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCreateKek(ctx, cmd.String("algorithm"))
				},
			},
			{
				Name:  "rotate-kek",
				Usage: "Rotate the Key Encryption Key (KEK)",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "algorithm",
						Aliases: []string{"alg"},
						Value:   "aes-gcm",
						Usage:   "Encryption algorithm to use (aes-gcm or chacha20-poly1305)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunRotateKek(ctx, cmd.String("algorithm"))
				},
			},
			{
				Name:  "create-client",
				Usage: "Create a new authentication client with policies",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    "Human-readable client name",
					},
					&cli.BoolFlag{
						Name:    "active",
						Aliases: []string{"a"},
						Value:   true,
						Usage:   "Whether the client can authenticate immediately",
					},
					&cli.StringFlag{
						Name:    "policies",
						Aliases: []string{"p"},
						Usage:   "JSON array of policy documents (omit for interactive mode)",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "text",
						Usage:   "Output format: 'text' or 'json'",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCreateClient(
						ctx,
						cmd.String("name"),
						cmd.Bool("active"),
						cmd.String("policies"),
						cmd.String("format"),
					)
				},
			},
			{
				Name:  "update-client",
				Usage: "Update an existing authentication client's configuration",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Aliases:  []string{"i"},
						Required: true,
						Usage:    "Client ID (UUID)",
					},
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    "Human-readable client name",
					},
					&cli.BoolFlag{
						Name:    "active",
						Aliases: []string{"a"},
						Value:   true,
						Usage:   "Whether the client can authenticate",
					},
					&cli.StringFlag{
						Name:    "policies",
						Aliases: []string{"p"},
						Usage:   "JSON array of policy documents (omit for interactive mode)",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "text",
						Usage:   "Output format: 'text' or 'json'",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunUpdateClient(
						ctx,
						cmd.String("id"),
						cmd.String("name"),
						cmd.Bool("active"),
						cmd.String("policies"),
						cmd.String("format"),
					)
				},
			},
			{
				Name:  "clean-audit-logs",
				Usage: "Delete audit logs older than specified days",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "days",
						Aliases:  []string{"d"},
						Required: true,
						Usage:    "Delete audit logs older than this many days",
					},
					&cli.BoolFlag{
						Name:    "dry-run",
						Aliases: []string{"n"},
						Value:   false,
						Usage:   "Show how many logs would be deleted without deleting",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "text",
						Usage:   "Output format: 'text' or 'json'",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCleanAuditLogs(
						ctx,
						cmd.Int("days"),
						cmd.Bool("dry-run"),
						cmd.String("format"),
					)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}
