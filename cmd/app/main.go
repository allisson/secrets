// Package main provides the entry point for the application with CLI commands.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/urfave/cli/v3"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// closeContainer closes all resources in the container and logs any errors.
func closeContainer(container *app.Container, logger *slog.Logger) {
	if err := container.Shutdown(context.Background()); err != nil {
		logger.Error("failed to shutdown container", slog.Any("error", err))
	}
}

// closeMigrate closes the migration instance and logs any errors.
func closeMigrate(migrate *migrate.Migrate, logger *slog.Logger) {
	sourceError, databaseError := migrate.Close()
	if sourceError != nil || databaseError != nil {
		logger.Error(
			"failed to close the migrate",
			slog.Any("source_error", sourceError),
			slog.Any("database_error", databaseError),
		)
	}
}

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
					return runServer(ctx)
				},
			},
			{
				Name:  "migrate",
				Usage: "Run database migrations",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runMigrations()
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
					return runCreateMasterKey(cmd.String("id"))
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
					return runCreateKek(ctx, cmd.String("algorithm"))
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
					return runRotateKek(ctx, cmd.String("algorithm"))
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}

// runServer starts the HTTP server with graceful shutdown support.
func runServer(ctx context.Context) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("starting server", slog.String("version", "1.0.0"))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get HTTP server from container (this initializes all dependencies)
	server, err := container.HTTPServer()
	if err != nil {
		return fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(ctx); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.DBConnMaxLifetime)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	case err := <-serverErr:
		return err
	}

	return nil
}

// runCreateMasterKey generates a new master key and displays the environment variable configuration.
//
// This command is a helper for generating cryptographically secure master keys for use in
// envelope encryption. The generated key is 32 bytes (256 bits) suitable for AES-256 encryption.
// This is the recommended way to generate master keys for the Secrets system.
//
// The key is generated using crypto/rand.Read which provides cryptographically secure random bytes.
// After encoding, the key material is immediately zeroed from memory for security. The output
// format matches the MASTER_KEYS and ACTIVE_MASTER_KEY_ID environment variables expected by
// LoadMasterKeyChainFromEnv.
//
// Parameters:
//   - keyID: Optional key ID for the master key (e.g., "prod-master-key-2025")
//     If empty, a default ID in the format "master-key-YYYY-MM-DD" is generated
//
// Output format:
//
//	MASTER_KEYS="<keyID>:<base64-encoded-key>"
//	ACTIVE_MASTER_KEY_ID="<keyID>"
//
// The command also displays helpful comments about key rotation with multiple keys.
//
// Workflow example:
//
//	# Step 1: Generate master key
//	./bin/app create-master-key --id prod-master-key-2025
//
//	# Step 2: Copy output to .env file
//	# MASTER_KEYS="prod-master-key-2025:..."
//	# ACTIVE_MASTER_KEY_ID="prod-master-key-2025"
//
//	# Step 3: Run migrations
//	./bin/app migrate
//
//	# Step 4: Create initial KEK
//	./bin/app create-kek
//
//	# Step 5: Start server
//	./bin/app server
//
// Security notes:
//   - Store the output securely (e.g., in a secrets manager or encrypted vault)
//   - Never commit master keys to version control
//   - For production, consider using a proper KMS instead of environment variables
//   - Rotate master keys periodically according to your security policy (e.g., every 90 days)
//   - When rotating, generate a new key, add it to MASTER_KEYS, update ACTIVE_MASTER_KEY_ID, and rotate KEKs
//
// Returns:
//   - An error if key generation fails
func runCreateMasterKey(keyID string) error {
	// Generate default key ID if not provided
	if keyID == "" {
		keyID = fmt.Sprintf("master-key-%s", time.Now().Format("2006-01-02"))
	}

	// Generate a cryptographically secure 32-byte master key
	masterKey := make([]byte, 32)
	if _, err := rand.Read(masterKey); err != nil {
		return fmt.Errorf("failed to generate master key: %w", err)
	}

	// Encode the master key to base64
	encodedKey := base64.StdEncoding.EncodeToString(masterKey)

	// Zero out the master key from memory for security
	for i := range masterKey {
		masterKey[i] = 0
	}

	// Print the environment variable configuration
	fmt.Println("# Master Key Configuration")
	fmt.Println("# Copy these environment variables to your .env file or secrets manager")
	fmt.Println()
	fmt.Printf("MASTER_KEYS=\"%s:%s\"\n", keyID, encodedKey)
	fmt.Printf("ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
	fmt.Println()
	fmt.Println("# For multiple master keys (key rotation), use comma-separated format:")
	fmt.Printf("# MASTER_KEYS=\"%s:%s,new-key:base64-encoded-new-key\"\n", keyID, encodedKey)
	fmt.Println("# ACTIVE_MASTER_KEY_ID=\"new-key\"")

	return nil
}

// runMigrations executes database migrations based on the configured driver.
func runMigrations() error {
	cfg := config.Load()

	// Create container just for logger
	container := app.NewContainer(cfg)
	logger := container.Logger()

	logger.Info("running database migrations",
		slog.String("driver", cfg.DBDriver),
	)

	// Determine migration path based on driver
	migrationsPath := "file://migrations/postgresql"
	if cfg.DBDriver == "mysql" {
		migrationsPath = "file://migrations/mysql"
	}

	m, err := migrate.New(migrationsPath, cfg.DBConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer closeMigrate(m, logger)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("migrations completed successfully")
	return nil
}

// runCreateKek creates a new Key Encryption Key using the specified algorithm.
//
// This command should only be run once during initial system setup to create the
// first KEK in the database. The KEK is encrypted using the active master key
// from the MASTER_KEYS environment variable.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - algorithmStr: The encryption algorithm ("aes-gcm" or "chacha20-poly1305")
//
// Requirements:
//   - Database must be migrated (run 'migrate' command first)
//   - MASTER_KEYS environment variable must be set
//   - ACTIVE_MASTER_KEY_ID environment variable must be set
//
// Returns:
//   - An error if the KEK already exists or creation fails
func runCreateKek(ctx context.Context, algorithmStr string) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("creating new KEK", slog.String("algorithm", algorithmStr))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Parse algorithm
	var algorithm cryptoDomain.Algorithm
	switch algorithmStr {
	case "aes-gcm":
		algorithm = cryptoDomain.AESGCM
	case "chacha20-poly1305":
		algorithm = cryptoDomain.ChaCha20
	default:
		return fmt.Errorf("invalid algorithm: %s (valid options: aes-gcm, chacha20-poly1305)", algorithmStr)
	}

	// Get master key chain from container
	masterKeyChain, err := container.MasterKeyChain()
	if err != nil {
		return fmt.Errorf("failed to load master key chain: %w", err)
	}

	logger.Info("master key chain loaded",
		slog.String("active_master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	// Get KEK use case from container
	kekUseCase, err := container.KekUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize KEK use case: %w", err)
	}

	// Create the KEK
	if err := kekUseCase.Create(ctx, masterKeyChain, algorithm); err != nil {
		return fmt.Errorf("failed to create KEK: %w", err)
	}

	logger.Info("KEK created successfully",
		slog.String("algorithm", string(algorithm)),
		slog.String("master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	return nil
}

// runRotateKek rotates the existing Key Encryption Key using the specified algorithm.
//
// This command creates a new KEK version and marks the previous active KEK as inactive.
// The new KEK is encrypted using the active master key from the MASTER_KEYS environment
// variable. This operation is atomic and maintains backward compatibility - existing
// DEKs encrypted with the old KEK remain readable.
//
// Key rotation is recommended every 90 days or when:
//   - Suspecting KEK compromise
//   - Changing encryption algorithms
//   - Rotating master keys
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - algorithmStr: The encryption algorithm for the new KEK ("aes-gcm" or "chacha20-poly1305")
//
// Requirements:
//   - An active KEK must already exist (run 'create-kek' first)
//   - MASTER_KEYS environment variable must be set
//   - ACTIVE_MASTER_KEY_ID environment variable must be set
//
// Returns:
//   - An error if no active KEK exists or rotation fails
func runRotateKek(ctx context.Context, algorithmStr string) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("rotating KEK", slog.String("algorithm", algorithmStr))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Parse algorithm
	var algorithm cryptoDomain.Algorithm
	switch algorithmStr {
	case "aes-gcm":
		algorithm = cryptoDomain.AESGCM
	case "chacha20-poly1305":
		algorithm = cryptoDomain.ChaCha20
	default:
		return fmt.Errorf("invalid algorithm: %s (valid options: aes-gcm, chacha20-poly1305)", algorithmStr)
	}

	// Get master key chain from container
	masterKeyChain, err := container.MasterKeyChain()
	if err != nil {
		return fmt.Errorf("failed to load master key chain: %w", err)
	}

	logger.Info("master key chain loaded",
		slog.String("active_master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	// Get KEK use case from container
	kekUseCase, err := container.KekUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize KEK use case: %w", err)
	}

	// Rotate the KEK
	if err := kekUseCase.Rotate(ctx, masterKeyChain, algorithm); err != nil {
		return fmt.Errorf("failed to rotate KEK: %w", err)
	}

	logger.Info("KEK rotated successfully",
		slog.String("algorithm", string(algorithm)),
		slog.String("master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	return nil
}
