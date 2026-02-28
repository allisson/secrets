package main

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/allisson/secrets/cmd/app/commands"
	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

func getAuthCommands() []*cli.Command {
	return []*cli.Command{
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
				cfg := config.Load()
				container := app.NewContainer(cfg)
				defer func() { _ = container.Shutdown(ctx) }()

				tokenizationUseCase, err := container.TokenizationUseCase()
				if err != nil {
					return err
				}

				return commands.RunCleanExpiredTokens(
					ctx,
					tokenizationUseCase,
					container.Logger(),
					commands.DefaultIO().Writer,
					int(cmd.Int("days")),
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
				cfg := config.Load()
				container := app.NewContainer(cfg)
				defer func() { _ = container.Shutdown(ctx) }()

				clientUseCase, err := container.ClientUseCase()
				if err != nil {
					return err
				}

				return commands.RunCreateClient(
					ctx,
					clientUseCase,
					container.Logger(),
					cmd.String("name"),
					cmd.Bool("active"),
					cmd.String("policies"),
					cmd.String("format"),
					commands.DefaultIO(),
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
				cfg := config.Load()
				container := app.NewContainer(cfg)
				defer func() { _ = container.Shutdown(ctx) }()

				clientUseCase, err := container.ClientUseCase()
				if err != nil {
					return err
				}

				return commands.RunUpdateClient(
					ctx,
					clientUseCase,
					container.Logger(),
					commands.DefaultIO(),
					cmd.String("id"),
					cmd.String("name"),
					cmd.Bool("active"),
					cmd.String("policies"),
					cmd.String("format"),
				)
			},
		},
	}
}
