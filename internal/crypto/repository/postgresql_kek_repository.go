// Package repository implements data persistence for cryptographic key management.
//
// This package provides repository implementations for storing and retrieving
// Key Encryption Keys (KEKs) and Data Encryption Keys (DEKs) in PostgreSQL and
// MySQL databases. Repositories follow the Repository pattern and support both
// direct database operations and transactional operations.
//
// # Key Components
//
// The package includes repositories for:
//   - KEK (Key Encryption Keys): Intermediate keys encrypted by master keys
//   - DEK (Data Encryption Keys): Keys used to encrypt application data
//
// # Database Support
//
// Each repository type has two implementations:
//   - PostgreSQL: Uses native UUID type and BYTEA for binary data
//   - MySQL: Uses BINARY(16) for UUIDs and BLOB for binary data
//
// # Transaction Support
//
// All repositories support transaction-aware operations via database.GetTx(),
// enabling atomic multi-step operations such as key rotation. When called within
// a transaction context, repositories automatically use the transaction connection.
//
// # Usage Example
//
//	// Create KEK repository
//	kekRepo := repository.NewPostgreSQLKekRepository(db)
//
//	// Use within a transaction
//	txManager := database.NewTxManager(db)
//	err := txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    return kekRepo.Create(txCtx, kek)
//	})
package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLKekRepository implements KEK persistence for PostgreSQL databases.
//
// This repository handles storing and retrieving Key Encryption Keys using
// PostgreSQL's native UUID type and BYTEA for binary data. It supports
// transaction-aware operations via database.GetTx(), enabling atomic key
// rotation operations.
//
// Database schema requirements:
//   - id: UUID PRIMARY KEY
//   - master_key_id: TEXT/VARCHAR (reference to master key)
//   - algorithm: TEXT/VARCHAR (e.g., "aes-gcm", "chacha20-poly1305")
//   - encrypted_key: BYTEA (encrypted KEK bytes)
//   - nonce: BYTEA (encryption nonce)
//   - version: INTEGER (for tracking KEK versions during rotation)
//   - created_at: TIMESTAMP WITH TIME ZONE
//
// Transaction support:
//
//	The repository automatically detects transaction context using database.GetTx().
//	All methods work both within and outside of transactions seamlessly.
//
// Example usage:
//
//	repo := NewPostgreSQLKekRepository(db)
//
//	// Create a KEK outside transaction
//	err := repo.Create(ctx, kek)
//
//	// Or within a transaction
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    // Both operations use the same transaction
//	    if err := repo.Update(txCtx, oldKek); err != nil {
//	        return err
//	    }
//	    return repo.Create(txCtx, newKek)
//	})
type PostgreSQLKekRepository struct {
	db *sql.DB
}

// Create inserts a new KEK into the PostgreSQL database.
//
// The KEK's ID is stored as a native UUID, and binary fields (EncryptedKey, Nonce)
// are stored as BYTEA. This method supports transaction context via database.GetTx(),
// enabling atomic multi-step operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - kek: The Key Encryption Key to insert (must have all required fields populated)
//
// Returns:
//   - An error if the insert fails (e.g., duplicate key, constraint violation)
//
// Example:
//
//	kek := &cryptoDomain.Kek{
//	    ID:           uuid.Must(uuid.NewV7()),
//	    MasterKeyID:  "master-key-1",
//	    Algorithm:    cryptoDomain.AESGCM,
//	    EncryptedKey: encryptedBytes,
//	    Nonce:        nonceBytes,
//	    Version:      1,
//	    CreatedAt:    time.Now().UTC(),
//	}
//	err := repo.Create(ctx, kek)
func (p *PostgreSQLKekRepository) Create(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := querier.ExecContext(
		ctx,
		query,
		kek.ID,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create kek")
	}
	return nil
}

// Update modifies an existing KEK in the PostgreSQL database.
//
// This method updates all mutable fields of the KEK. It supports transaction
// context via database.GetTx(), enabling atomic rotation operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - kek: The KEK with updated field values (ID must match existing record)
//
// Returns:
//   - An error if the update fails (e.g., KEK not found, constraint violation)
//
// Example:
//
//	// Update KEK during rotation
//	err := repo.Update(ctx, kek)
func (p *PostgreSQLKekRepository) Update(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE keks 
			  SET master_key_id = $1, 
			  	  algorithm = $2,
				  encrypted_key = $3,
				  nonce = $4,
				  version = $5, 
				  created_at = $6
			  WHERE id = $7`

	_, err := querier.ExecContext(
		ctx,
		query,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
		kek.ID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update kek")
	}

	return nil
}

// List retrieves all KEKs from the PostgreSQL database ordered by version descending.
//
// This method returns all KEKs (both active and inactive) sorted by version
// in descending order, ensuring the newest KEK appears first. This ordering
// is critical for key rotation scenarios where you need to identify the latest
// KEK version to set as active in the KekChain.
//
// The method supports transaction context via database.GetTx(), allowing
// consistent reads within a transaction.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//
// Returns:
//   - A slice of KEK pointers ordered by version descending (newest first)
//   - An error if the query fails
//
// Example:
//
//	// Load all KEKs for creating a KekChain
//	keks, err := repo.List(ctx)
//	if err != nil {
//	    return nil, err
//	}
//	if len(keks) == 0 {
//	    return nil, errors.New("no KEKs found")
//	}
//
//	// First KEK is the newest (highest version)
//	kekChain := cryptoDomain.NewKekChain(keks)
func (p *PostgreSQLKekRepository) List(ctx context.Context) ([]*cryptoDomain.Kek, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT * FROM keks ORDER BY version DESC`

	rows, err := querier.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var keks []*cryptoDomain.Kek
	for rows.Next() {
		var kek cryptoDomain.Kek

		err := rows.Scan(
			&kek.ID,
			&kek.MasterKeyID,
			&kek.Algorithm,
			&kek.EncryptedKey,
			&kek.Nonce,
			&kek.Version,
			&kek.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		keks = append(keks, &kek)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keks, nil
}

// NewPostgreSQLKekRepository creates a new PostgreSQL KEK repository instance.
//
// Parameters:
//   - db: A PostgreSQL database connection
//
// Returns:
//   - A new PostgreSQLKekRepository ready for use
//
// Example:
//
//	db, err := sql.Open("postgres", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewPostgreSQLKekRepository(db)
func NewPostgreSQLKekRepository(db *sql.DB) *PostgreSQLKekRepository {
	return &PostgreSQLKekRepository{db: db}
}
