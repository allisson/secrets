// Package main provides the CLI command definitions for the application.
package main

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/allisson/secrets/cmd/app/commands"
	"github.com/allisson/secrets/internal/app"
)

// getSystemCommands returns the system-related CLI commands.
func getSystemCommands(version string) []*cli.Command {
	return []*cli.Command{
		{
			Name:  "server",
			Usage: "Start the HTTP server",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.RunServer(ctx, version)
			},
		},
		{
			Name:  "migrate",
			Usage: "Run database migrations",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						cfg := container.Config()
						return commands.RunMigrations(
							container.Logger(),
							cfg.DBDriver,
							cfg.DBConnectionString,
						)
					},
				)
			},
		},
		{
			Name:  "migrate-down",
			Usage: "Rollback database migrations",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "steps",
					Aliases: []string{"n"},
					Value:   1,
					Usage:   "Number of migrations to rollback",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						cfg := container.Config()
						return commands.RunMigrationsDown(
							container.Logger(),
							cfg.DBDriver,
							cfg.DBConnectionString,
							int(cmd.Int("steps")),
						)
					},
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
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						auditLogUseCase, err := container.AuditLogUseCase(ctx)
						if err != nil {
							return err
						}

						return commands.RunCleanAuditLogs(
							ctx,
							auditLogUseCase,
							container.Logger(),
							commands.DefaultIO().Writer,
							int(cmd.Int("days")),
							cmd.Bool("dry-run"),
							cmd.String("format"),
						)
					},
				)
			},
		},
		{
			Name:  "purge-secrets",
			Usage: "Permanently delete soft-deleted secrets older than specified days",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "days",
					Aliases: []string{"d"},
					Value:   30,
					Usage:   "Delete secrets soft-deleted more than this many days ago",
				},
				&cli.BoolFlag{
					Name:    "dry-run",
					Aliases: []string{"n"},
					Value:   false,
					Usage:   "Show how many secrets would be deleted without deleting",
				},
				&cli.StringFlag{
					Name:    "format",
					Aliases: []string{"f"},
					Value:   "text",
					Usage:   "Output format: 'text' or 'json'",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						secretUseCase, err := container.SecretUseCase(ctx)
						if err != nil {
							return err
						}

						return commands.RunPurgeSecrets(
							ctx,
							secretUseCase,
							container.Logger(),
							commands.DefaultIO().Writer,
							int(cmd.Int("days")),
							cmd.Bool("dry-run"),
							cmd.String("format"),
						)
					},
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
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						auditLogUseCase, err := container.AuditLogUseCase(ctx)
						if err != nil {
							return err
						}

						return commands.RunVerifyAuditLogs(
							ctx,
							auditLogUseCase,
							container.Logger(),
							commands.DefaultIO().Writer,
							cmd.String("start-date"),
							cmd.String("end-date"),
							cmd.String("format"),
						)
					},
				)
			},
		},
	}
}
