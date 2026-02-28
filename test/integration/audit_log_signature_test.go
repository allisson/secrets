// Package integration provides integration tests for audit log cryptographic signatures.
package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/app"
	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/testutil"
)

// TestAuditLogSignature_EndToEnd verifies complete audit log signing and verification workflow.
func TestAuditLogSignature_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dbConfigs := []struct {
		name   string
		driver string
		dsn    string
	}{
		{
			name:   "PostgreSQL",
			driver: "postgres",
			dsn:    testutil.GetPostgresTestDSN(),
		},
		{
			name:   "MySQL",
			driver: "mysql",
			dsn:    testutil.GetMySQLTestDSN(),
		},
	}

	for _, dbConfig := range dbConfigs {
		t.Run(dbConfig.name, func(t *testing.T) {
			ctx := context.Background()
			driver := dbConfig.driver // Capture driver for inner test functions

			// Setup test database and dependencies
			testCtx := setupAuditLogTestContext(t, driver, dbConfig.dsn)
			defer cleanupAuditLogTestContext(t, testCtx)

			// Create audit signer and load KEK chain
			auditSigner := authService.NewAuditSigner()
			kekChain := testCtx.kekChain

			// Get repositories from container
			auditLogRepo, err := testCtx.container.AuditLogRepository()
			require.NoError(t, err, "failed to get audit log repository")

			// Create use case with signing enabled
			auditLogUseCase := authUseCase.NewAuditLogUseCase(auditLogRepo, auditSigner, kekChain)

			t.Run("CreateSignedAuditLog", func(t *testing.T) {
				// Create a signed audit log
				requestID := uuid.Must(uuid.NewV7())
				clientID := testCtx.rootClient.ID
				capability := authDomain.ReadCapability
				path := "/api/v1/secrets/test-key"
				metadata := map[string]any{
					"user_agent": "integration-test",
					"ip_address": "127.0.0.1",
				}

				err := auditLogUseCase.Create(ctx, requestID, clientID, capability, path, metadata)
				require.NoError(t, err, "failed to create audit log")

				// Retrieve the created log
				logs, err := auditLogUseCase.List(ctx, 0, 1, nil, nil)
				require.NoError(t, err, "failed to list audit logs")
				require.Len(t, logs, 1, "expected exactly one audit log")

				log := logs[0]

				// Verify signature fields are populated
				assert.True(t, log.IsSigned, "audit log should be signed")
				assert.NotNil(t, log.KekID, "kek_id should not be nil")
				assert.NotEmpty(t, log.Signature, "signature should not be empty")
				assert.Equal(t, kekChain.ActiveKekID(), *log.KekID, "kek_id should match active KEK")

				// Verify the signature is valid
				err = auditLogUseCase.VerifyIntegrity(ctx, log.ID)
				assert.NoError(t, err, "signature verification should succeed")
			})

			t.Run("TamperDetection", func(t *testing.T) {
				// Create a signed audit log
				requestID := uuid.Must(uuid.NewV7())
				clientID := testCtx.rootClient.ID

				err := auditLogUseCase.Create(
					ctx,
					requestID,
					clientID,
					authDomain.WriteCapability,
					"/api/v1/secrets/tamper-test",
					nil,
				)
				require.NoError(t, err, "failed to create audit log")

				// Retrieve the log
				logs, err := auditLogUseCase.List(ctx, 0, 1, nil, nil)
				require.NoError(t, err, "failed to list audit logs")
				require.Len(t, logs, 1, "expected exactly one audit log")

				log := logs[0]

				// Tamper with the log by modifying the path directly in the database
				var execErr error
				var result sql.Result
				if driver == "postgres" {
					result, execErr = testCtx.db.Exec(
						"UPDATE audit_logs SET path = '/api/v1/secrets/tampered' WHERE id = $1",
						log.ID,
					)
				} else {
					// MySQL stores UUID as BINARY(16), need binary representation
					idBinary, marshalErr := log.ID.MarshalBinary()
					require.NoError(t, marshalErr, "failed to marshal UUID")
					result, execErr = testCtx.db.Exec(
						"UPDATE audit_logs SET path = '/api/v1/secrets/tampered' WHERE id = ?",
						idBinary,
					)
				}
				require.NoError(t, execErr, "failed to tamper with audit log")

				// Verify the UPDATE actually modified a row
				rowsAffected, _ := result.RowsAffected()
				require.Equal(t, int64(1), rowsAffected, "UPDATE should affect exactly 1 row")

				// Verification should now fail
				err = auditLogUseCase.VerifyIntegrity(ctx, log.ID)
				assert.Error(t, err, "signature verification should fail for tampered log")
				assert.ErrorIs(t, err, authDomain.ErrSignatureInvalid, "error should be ErrSignatureInvalid")
			})

			t.Run("VerifyBatch_AllValid", func(t *testing.T) {
				// Create multiple signed audit logs
				startTime := time.Now().UTC()
				clientID := testCtx.rootClient.ID

				for i := 0; i < 5; i++ {
					requestID := uuid.Must(uuid.NewV7())
					path := "/api/v1/secrets/batch-test-" + string(rune('a'+i))

					err := auditLogUseCase.Create(
						ctx,
						requestID,
						clientID,
						authDomain.ReadCapability,
						path,
						nil,
					)
					require.NoError(t, err, "failed to create audit log")

					time.Sleep(10 * time.Millisecond) // Ensure distinct timestamps
				}

				endTime := time.Now().UTC().Add(1 * time.Second)

				// Verify batch
				report, err := auditLogUseCase.VerifyBatch(ctx, startTime, endTime)
				require.NoError(t, err, "batch verification should succeed")

				assert.Equal(t, int64(5), report.TotalChecked, "should check 5 logs")
				assert.Equal(t, int64(5), report.SignedCount, "all 5 should be signed")
				assert.Equal(t, int64(5), report.ValidCount, "all 5 should be valid")
				assert.Equal(t, int64(0), report.InvalidCount, "no invalid logs")
				assert.Empty(t, report.InvalidLogs, "no invalid log IDs")
			})

			t.Run("VerifyBatch_WithInvalid", func(t *testing.T) {
				// Create signed audit logs
				startTime := time.Now().UTC()
				clientID := testCtx.rootClient.ID

				var logIDs []uuid.UUID
				for i := 0; i < 3; i++ {
					requestID := uuid.Must(uuid.NewV7())
					path := "/api/v1/secrets/invalid-test-" + string(rune('a'+i))

					err := auditLogUseCase.Create(
						ctx,
						requestID,
						clientID,
						authDomain.WriteCapability,
						path,
						nil,
					)
					require.NoError(t, err, "failed to create audit log")

					time.Sleep(10 * time.Millisecond)
				}

				// Get the created logs
				endTime := time.Now().UTC().Add(1 * time.Second)
				logs, err := auditLogUseCase.List(ctx, 0, 3, &startTime, &endTime)
				require.NoError(t, err, "failed to list audit logs")
				require.Len(t, logs, 3, "expected 3 audit logs")

				for _, log := range logs {
					logIDs = append(logIDs, log.ID)
				}

				// Tamper with the middle log
				var execErr error
				if driver == "postgres" {
					_, execErr = testCtx.db.Exec(
						"UPDATE audit_logs SET capability = 'delete' WHERE id = $1",
						logIDs[1],
					)
				} else {
					// MySQL stores UUID as BINARY(16), need binary representation
					idBinary, marshalErr := logIDs[1].MarshalBinary()
					require.NoError(t, marshalErr, "failed to marshal UUID")
					_, execErr = testCtx.db.Exec(
						"UPDATE audit_logs SET capability = 'delete' WHERE id = ?",
						idBinary,
					)
				}
				require.NoError(t, execErr, "failed to tamper with audit log")

				// Verify batch
				report, err := auditLogUseCase.VerifyBatch(ctx, startTime, endTime)
				require.NoError(t, err, "batch verification should not error")

				assert.Equal(t, int64(3), report.TotalChecked, "should check 3 logs")
				assert.Equal(t, int64(3), report.SignedCount, "all 3 should be signed")
				assert.Equal(t, int64(2), report.ValidCount, "2 should be valid")
				assert.Equal(t, int64(1), report.InvalidCount, "1 should be invalid")
				assert.Len(t, report.InvalidLogs, 1, "should have 1 invalid log ID")
				assert.Equal(t, logIDs[1], report.InvalidLogs[0], "invalid log ID should match tampered log")
			})

			t.Run("LegacyUnsignedLogs", func(t *testing.T) {
				// Create an unsigned legacy audit log (using nil signer and chain)
				legacyUseCase := authUseCase.NewAuditLogUseCase(auditLogRepo, nil, nil)

				requestID := uuid.Must(uuid.NewV7())
				clientID := testCtx.rootClient.ID

				err := legacyUseCase.Create(
					ctx,
					requestID,
					clientID,
					authDomain.ReadCapability,
					"/api/v1/secrets/legacy",
					nil,
				)
				require.NoError(t, err, "failed to create legacy audit log")

				// Retrieve the log
				logs, err := legacyUseCase.List(ctx, 0, 1, nil, nil)
				require.NoError(t, err, "failed to list audit logs")
				require.Len(t, logs, 1, "expected exactly one audit log")

				log := logs[0]

				// Verify it's unsigned
				assert.False(t, log.IsSigned, "audit log should not be signed")
				assert.Nil(t, log.KekID, "kek_id should be nil")
				assert.Empty(t, log.Signature, "signature should be empty")

				// Verification should return ErrSignatureMissing
				err = auditLogUseCase.VerifyIntegrity(ctx, log.ID)
				assert.Error(t, err, "verification should fail for unsigned log")
				assert.ErrorIs(t, err, authDomain.ErrSignatureMissing, "error should be ErrSignatureMissing")
			})

			t.Run("VerifyBatch_MixedSignedAndLegacy", func(t *testing.T) {
				startTime := time.Now().UTC()
				clientID := testCtx.rootClient.ID

				// Create 2 signed logs
				signedUseCase := authUseCase.NewAuditLogUseCase(auditLogRepo, auditSigner, kekChain)
				for i := 0; i < 2; i++ {
					requestID := uuid.Must(uuid.NewV7())
					err := signedUseCase.Create(
						ctx,
						requestID,
						clientID,
						authDomain.ReadCapability,
						"/signed",
						nil,
					)
					require.NoError(t, err)
					time.Sleep(10 * time.Millisecond)
				}

				// Create 2 unsigned legacy logs
				legacyUseCase := authUseCase.NewAuditLogUseCase(auditLogRepo, nil, nil)
				for i := 0; i < 2; i++ {
					requestID := uuid.Must(uuid.NewV7())
					err := legacyUseCase.Create(
						ctx,
						requestID,
						clientID,
						authDomain.WriteCapability,
						"/legacy",
						nil,
					)
					require.NoError(t, err)
					time.Sleep(10 * time.Millisecond)
				}

				endTime := time.Now().UTC().Add(1 * time.Second)

				// Verify batch
				report, err := signedUseCase.VerifyBatch(ctx, startTime, endTime)
				require.NoError(t, err, "batch verification should succeed")

				assert.Equal(t, int64(4), report.TotalChecked, "should check 4 logs")
				assert.Equal(t, int64(2), report.SignedCount, "2 should be signed")
				assert.Equal(t, int64(2), report.UnsignedCount, "2 should be unsigned")
				assert.Equal(t, int64(2), report.ValidCount, "2 signed should be valid")
				assert.Equal(t, int64(0), report.InvalidCount, "no invalid logs")
			})
		})
	}
}

// auditLogTestContext holds test dependencies for audit log signature tests.
type auditLogTestContext struct {
	container  *app.Container
	db         *sql.DB
	kekChain   *cryptoDomain.KekChain
	rootClient *authDomain.Client
}

// setupAuditLogTestContext creates a test environment with database, KEK chain, and root client.
func setupAuditLogTestContext(t *testing.T, driver, dsn string) *auditLogTestContext {
	t.Helper()

	// Initialize test database with migrations
	var db *sql.DB
	if driver == "postgres" {
		db = testutil.SetupPostgresDB(t)
	} else {
		db = testutil.SetupMySQLDB(t)
	}

	// Generate ephemeral master key and create chain
	masterKey := generateMasterKey()
	masterKeyChain := createMasterKeyChain(masterKey)

	// Create config with database settings
	cfg := &config.Config{
		DBDriver:             driver,
		DBConnectionString:   dsn,
		DBMaxOpenConnections: 10,
		DBMaxIdleConnections: 5,
		DBConnMaxLifetime:    time.Hour,
		LogLevel:             "error",
		MetricsEnabled:       false,
		ServerPort:           8080,
		KMSProvider:          "",
		KMSKeyURI:            "",
		AuthTokenExpiration:  24 * time.Hour,
	}

	// Create DI container
	container := app.NewContainer(cfg)

	// Create initial KEK for signing
	ctx := context.Background()
	kekUseCase, err := container.KekUseCase()
	require.NoError(t, err, "failed to get kek use case")

	err = kekUseCase.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)
	require.NoError(t, err, "failed to create KEK")

	// Load KEK chain
	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
	require.NoError(t, err, "failed to unwrap KEK chain")

	// Create root client for test operations
	clientUseCase, err := container.ClientUseCase()
	require.NoError(t, err, "failed to get client use case")

	rootPolicies := []authDomain.PolicyDocument{
		{
			Path: "*",
			Capabilities: []authDomain.Capability{
				authDomain.ReadCapability,
				authDomain.WriteCapability,
				authDomain.DeleteCapability,
			},
		},
	}

	createInput := &authDomain.CreateClientInput{
		Name:     "integration-test-root",
		IsActive: true,
		Policies: rootPolicies,
	}

	output, err := clientUseCase.Create(ctx, createInput)
	require.NoError(t, err, "failed to create root client")

	// Retrieve the created client
	rootClient, err := clientUseCase.Get(ctx, output.ID)
	require.NoError(t, err, "failed to get root client")

	return &auditLogTestContext{
		container:  container,
		db:         db,
		kekChain:   kekChain,
		rootClient: rootClient,
	}
}

// cleanupAuditLogTestContext closes database and container resources.
func cleanupAuditLogTestContext(t *testing.T, testCtx *auditLogTestContext) {
	t.Helper()

	if err := testCtx.container.Shutdown(context.Background()); err != nil {
		t.Logf("Warning: failed to shutdown container: %v", err)
	}

	if err := testCtx.db.Close(); err != nil {
		t.Logf("Warning: failed to close database: %v", err)
	}
}
