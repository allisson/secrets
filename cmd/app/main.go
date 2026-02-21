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
		Version: "0.8.0",
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
					&cli.StringFlag{
						Name:  "kms-provider",
						Value: "",
						Usage: "KMS provider (localsecrets, gcpkms, awskms, azurekeyvault, hashivault)",
					},
					&cli.StringFlag{
						Name:  "kms-key-uri",
						Value: "",
						Usage: "KMS key URI (e.g., base64key://, gcpkms://projects/.../cryptoKeys/...)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCreateMasterKey(
						cmd.String("id"),
						cmd.String("kms-provider"),
						cmd.String("kms-key-uri"),
					)
				},
			},
			{
				Name:  "rotate-master-key",
				Usage: "Rotate the Master Key by generating a new key and combining with existing keys",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "id",
						Aliases: []string{"i"},
						Value:   "",
						Usage:   "New master key ID (e.g., prod-master-key-2026)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunRotateMasterKey(ctx, cmd.String("id"))
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
				Name:  "create-tokenization-key",
				Usage: "Create a new tokenization key for format-preserving tokens",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    "Unique name for the tokenization key",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"fmt"},
						Value:   "uuid",
						Usage:   "Token format: uuid, numeric, luhn-preserving, or alphanumeric",
					},
					&cli.BoolFlag{
						Name:    "deterministic",
						Aliases: []string{"det"},
						Value:   false,
						Usage:   "Enable deterministic mode (same plaintext → same token)",
					},
					&cli.StringFlag{
						Name:    "algorithm",
						Aliases: []string{"alg"},
						Value:   "aes-gcm",
						Usage:   "Encryption algorithm to use (aes-gcm or chacha20-poly1305)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCreateTokenizationKey(
						ctx,
						cmd.String("name"),
						cmd.String("format"),
						cmd.Bool("deterministic"),
						cmd.String("algorithm"),
					)
				},
			},
			{
				Name:  "rotate-tokenization-key",
				Usage: "Rotate an existing tokenization key to a new version",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    "Name of the tokenization key to rotate",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"fmt"},
						Value:   "uuid",
						Usage:   "Token format: uuid, numeric, luhn-preserving, or alphanumeric",
					},
					&cli.BoolFlag{
						Name:    "deterministic",
						Aliases: []string{"det"},
						Value:   false,
						Usage:   "Enable deterministic mode (same plaintext → same token)",
					},
					&cli.StringFlag{
						Name:    "algorithm",
						Aliases: []string{"alg"},
						Value:   "aes-gcm",
						Usage:   "Encryption algorithm to use (aes-gcm or chacha20-poly1305)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunRotateTokenizationKey(
						ctx,
						cmd.String("name"),
						cmd.String("format"),
						cmd.Bool("deterministic"),
						cmd.String("algorithm"),
					)
				},
			},
			{
				Name:  "clean-expired-tokens",
				Usage: "Delete expired tokens older than specified days",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "days",
						Aliases:  []string{"d"},
						Required: true,
						Usage:    "Delete expired tokens older than this many days",
					},
					&cli.BoolFlag{
						Name:    "dry-run",
						Aliases: []string{"n"},
						Value:   false,
						Usage:   "Show how many tokens would be deleted without deleting",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "text",
						Usage:   "Output format: 'text' or 'json'",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunCleanExpiredTokens(
						ctx,
						cmd.Int("days"),
						cmd.Bool("dry-run"),
						cmd.String("format"),
					)
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
			{
				Name:  "verify-audit-logs",
				Usage: "Verify cryptographic integrity of audit logs",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "start-date",
						Aliases:  []string{"s"},
						Required: true,
						Usage:    "Start date in YYYY-MM-DD or YYYY-MM-DD HH:MM:SS format",
					},
					&cli.StringFlag{
						Name:     "end-date",
						Aliases:  []string{"e"},
						Required: true,
						Usage:    "End date in YYYY-MM-DD or YYYY-MM-DD HH:MM:SS format",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "text",
						Usage:   "Output format: 'text' or 'json'",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return commands.RunVerifyAuditLogs(
						ctx,
						cmd.String("start-date"),
						cmd.String("end-date"),
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
