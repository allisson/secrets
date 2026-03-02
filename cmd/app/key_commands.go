// Package main provides the CLI command definitions for the application.
package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/allisson/secrets/cmd/app/commands"
	"github.com/allisson/secrets/internal/app"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
)

// getKeyCommands returns the key-related CLI commands.
func getKeyCommands() []*cli.Command {
	return []*cli.Command{
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
					Name:     "kms-provider",
					Value:    "",
					Required: true,
					Usage:    "KMS provider (localsecrets, gcpkms, awskms, azurekeyvault, hashivault)",
				},
				&cli.StringFlag{
					Name:     "kms-key-uri",
					Value:    "",
					Required: true,
					Usage:    "KMS key URI (e.g., base64key://, gcpkms://projects/.../cryptoKeys/...)",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						return commands.RunCreateMasterKey(
							ctx,
							cryptoService.NewKMSService(),
							container.Logger(),
							commands.DefaultIO().Writer,
							cmd.String("id"),
							cmd.String("kms-provider"),
							cmd.String("kms-key-uri"),
						)
					},
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
				&cli.StringFlag{
					Name:     "kms-provider",
					Value:    "",
					Required: true,
					Usage:    "KMS provider (localsecrets, gcpkms, awskms, azurekeyvault, hashivault)",
				},
				&cli.StringFlag{
					Name:     "kms-key-uri",
					Value:    "",
					Required: true,
					Usage:    "KMS key URI (e.g., base64key://, gcpkms://projects/.../cryptoKeys/...)",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						return commands.RunRotateMasterKey(
							ctx,
							cryptoService.NewKMSService(),
							container.Logger(),
							commands.DefaultIO().Writer,
							cmd.String("id"),
							cmd.String("kms-provider"),
							cmd.String("kms-key-uri"),
							os.Getenv("MASTER_KEYS"),
							os.Getenv("ACTIVE_MASTER_KEY_ID"),
						)
					},
				)
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
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						kekUseCase, err := container.KekUseCase(ctx)
						if err != nil {
							return err
						}

						masterKeyChain, err := container.MasterKeyChain(ctx)
						if err != nil {
							return err
						}

						return commands.RunCreateKek(
							ctx,
							kekUseCase,
							masterKeyChain,
							container.Logger(),
							cmd.String("algorithm"),
						)
					},
				)
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
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						kekUseCase, err := container.KekUseCase(ctx)
						if err != nil {
							return err
						}

						masterKeyChain, err := container.MasterKeyChain(ctx)
						if err != nil {
							return err
						}

						return commands.RunRotateKek(
							ctx,
							kekUseCase,
							masterKeyChain,
							container.Logger(),
							cmd.String("algorithm"),
						)
					},
				)
			},
		},
		{
			Name:  "rewrap-deks",
			Usage: "Rewrap all DEKs that are not encrypted with a specific KEK",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "kek-id",
					Aliases:  []string{"k"},
					Required: true,
					Usage:    "Target KEK ID to encrypt the DEKs with",
				},
				&cli.IntFlag{
					Name:    "batch-size",
					Aliases: []string{"b"},
					Value:   100,
					Usage:   "Number of DEKs to process per batch",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						masterKeyChain, err := container.MasterKeyChain(ctx)
						if err != nil {
							return err
						}

						kekUseCase, err := container.KekUseCase(ctx)
						if err != nil {
							return err
						}

						dekUseCase, err := container.CryptoDekUseCase(ctx)
						if err != nil {
							return err
						}

						return commands.RunRewrapDeks(
							ctx,
							masterKeyChain,
							kekUseCase,
							dekUseCase,
							container.Logger(),
							cmd.String("kek-id"),
							int(cmd.Int("batch-size")),
						)
					},
				)
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
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						tokenizationKeyUseCase, err := container.TokenizationKeyUseCase(ctx)
						if err != nil {
							return err
						}

						return commands.RunCreateTokenizationKey(
							ctx,
							tokenizationKeyUseCase,
							container.Logger(),
							cmd.String("name"),
							cmd.String("format"),
							cmd.Bool("deterministic"),
							cmd.String("algorithm"),
						)
					},
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
				return commands.ExecuteWithContainer(
					ctx,
					func(ctx context.Context, container *app.Container) error {
						tokenizationKeyUseCase, err := container.TokenizationKeyUseCase(ctx)
						if err != nil {
							return err
						}

						return commands.RunRotateTokenizationKey(
							ctx,
							tokenizationKeyUseCase,
							container.Logger(),
							cmd.String("name"),
							cmd.String("format"),
							cmd.Bool("deterministic"),
							cmd.String("algorithm"),
						)
					},
				)
			},
		},
	}
}
