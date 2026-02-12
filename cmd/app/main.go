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
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}
