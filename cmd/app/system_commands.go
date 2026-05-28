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
						bm, err := container.BusinessMetrics(ctx)
						if err != nil {
							return err
						}

						return commands.RunRetentionSweep(
							ctx,
							commands.SweepSpec{
								Verb:           "delete",
								VerbPast:       "deleted",
								Subject:        "audit log(s)",
								MetricModule:   "auth",
								MetricOp:       "audit_log_clean",
								SupportsDryRun: true,
								Sweep: func(c context.Context, days int, dryRun bool) (int64, error) {
									return auditLogUseCase.DeleteOlderThan(c, days, dryRun)
								},
							},
							bm,
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
						bm, err := container.BusinessMetrics(ctx)
						if err != nil {
							return err
						}

						return commands.RunRetentionSweep(
							ctx,
							commands.SweepSpec{
								Verb:           "delete",
								VerbPast:       "deleted",
								Subject:        "secret(s)",
								MetricModule:   "secrets",
								MetricOp:       "secret_purge_deleted",
								SupportsDryRun: true,
								Sweep: func(c context.Context, days int, dryRun bool) (int64, error) {
									return secretUseCase.PurgeDeleted(c, days, dryRun)
								},
							},
							bm,
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
			Name:  "purge-transit-keys",
			Usage: "Permanently delete soft-deleted transit keys older than specified days",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "days",
					Aliases: []string{"d"},
					Value:   30,
					Usage:   "Delete transit keys soft-deleted more than this many days ago",
				},
				&cli.BoolFlag{
					Name:    "dry-run",
					Aliases: []string{"n"},
					Value:   false,
					Usage:   "Show how many transit keys would be deleted without deleting",
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
						transitUseCase, err := container.TransitKeyUseCase(ctx)
						if err != nil {
							return err
						}
						bm, err := container.BusinessMetrics(ctx)
						if err != nil {
							return err
						}

						return commands.RunRetentionSweep(
							ctx,
							commands.SweepSpec{
								Verb:           "delete",
								VerbPast:       "deleted",
								Subject:        "transit key(s)",
								MetricModule:   "transit",
								MetricOp:       "transit_key_purge_deleted",
								SupportsDryRun: true,
								Sweep: func(c context.Context, days int, dryRun bool) (int64, error) {
									return transitUseCase.PurgeDeleted(c, days, dryRun)
								},
							},
							bm,
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
			Name:  "purge-tokenization-keys",
			Usage: "Permanently delete soft-deleted tokenization keys and associated tokens older than specified days",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "days",
					Aliases: []string{"d"},
					Value:   30,
					Usage:   "Delete tokenization keys soft-deleted more than this many days ago",
				},
				&cli.BoolFlag{
					Name:    "dry-run",
					Aliases: []string{"n"},
					Value:   false,
					Usage:   "Show how many tokenization keys would be deleted without deleting",
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
						tokenizationUseCase, err := container.TokenizationKeyUseCase(ctx)
						if err != nil {
							return err
						}
						bm, err := container.BusinessMetrics(ctx)
						if err != nil {
							return err
						}

						return commands.RunRetentionSweep(
							ctx,
							commands.SweepSpec{
								Verb:           "delete",
								VerbPast:       "deleted",
								Subject:        "tokenization key(s) (and associated tokens)",
								MetricModule:   "tokenization",
								MetricOp:       "tokenization_key_purge_deleted",
								SupportsDryRun: true,
								Sweep: func(c context.Context, days int, dryRun bool) (int64, error) {
									return tokenizationUseCase.PurgeDeleted(c, days, dryRun)
								},
							},
							bm,
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
