// Package repository implements data persistence for cryptographic key management.
//
// This package provides repository implementations for storing and retrieving
// Key Encryption Keys (KEKs) and Data Encryption Keys (DEKs) in PostgreSQL and
// MySQL databases. Repositories follow the Repository pattern and support both
// direct database operations and transactional operations.
//
// The package includes repositories for:
//   - KEK (Key Encryption Keys): Intermediate keys encrypted by master keys
//   - DEK (Data Encryption Keys): Keys used to encrypt application data
//
// All repositories support transaction-aware operations via database.GetTx(),
// enabling atomic multi-step operations such as key rotation.
package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLDekRepository implements DEK persistence for PostgreSQL databases.
//
// This repository handles storing and retrieving Data Encryption Keys using
// PostgreSQL's native UUID type and BYTEA for binary data. It supports
// transaction-aware operations via database.GetTx(), enabling atomic operations
// such as DEK creation and updates during key rotation.
//
// Database schema requirements:
//   - id: UUID PRIMARY KEY
//   - kek_id: UUID FOREIGN KEY (reference to KEK)
//   - algorithm: TEXT/VARCHAR (e.g., "aes-gcm", "chacha20-poly1305")
//   - encrypted_key: BYTEA (encrypted DEK bytes)
//   - nonce: BYTEA (encryption nonce)
//   - created_at: TIMESTAMP WITH TIME ZONE
//
// Transaction support:
//
//	The repository automatically detects transaction context using database.GetTx().
//	All methods work both within and outside of transactions seamlessly.
//
// Example usage:
//
//	repo := NewPostgreSQLDekRepository(db)
//
//	// Create a DEK outside transaction
//	err := repo.Create(ctx, dek)
//
//	// Or within a transaction
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    // Both operations use the same transaction
//	    if err := repo.Update(txCtx, oldDek); err != nil {
//	        return err
//	    }
//	    return repo.Create(txCtx, newDek)
//	})
type PostgreSQLDekRepository struct {
	db *sql.DB
}

// Create inserts a new DEK into the PostgreSQL database.
//
// The DEK's ID is stored as a native UUID, and binary fields (EncryptedKey, Nonce)
// are stored as BYTEA. This method supports transaction context via database.GetTx(),
// enabling atomic multi-step operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - dek: The Data Encryption Key to insert (must have all required fields populated)
//
// Returns:
//   - An error if the insert fails (e.g., duplicate key, constraint violation)
//
// Example:
//
//	dek := &cryptoDomain.Dek{
//	    ID:           uuid.Must(uuid.NewV7()),
//	    KekID:        kekID,
//	    Algorithm:    cryptoDomain.AESGCM,
//	    EncryptedKey: encryptedBytes,
//	    Nonce:        nonceBytes,
//	    CreatedAt:    time.Now().UTC(),
//	}
//	err := repo.Create(ctx, dek)
func (p *PostgreSQLDekRepository) Create(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := querier.ExecContext(
		ctx,
		query,
		dek.ID,
		dek.KekID,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create dek")
	}
	return nil
}

// Update modifies an existing DEK in the PostgreSQL database.
//
// This method updates all mutable fields of the DEK. It supports transaction
// context via database.GetTx(), enabling atomic operations such as re-encrypting
// a DEK with a new KEK during key rotation.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - dek: The DEK with updated field values (ID must match existing record)
//
// Returns:
//   - An error if the update fails (e.g., DEK not found, constraint violation)
//
// Example:
//
//	// Re-encrypt DEK with a new KEK
//	dek.KekID = newKekID
//	dek.EncryptedKey = newEncryptedBytes
//	dek.Nonce = newNonce
//	err := repo.Update(ctx, dek)
func (p *PostgreSQLDekRepository) Update(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE deks 
			  SET kek_id = $1, 
			  	  algorithm = $2,
				  encrypted_key = $3,
				  nonce = $4,
				  created_at = $5
			  WHERE id = $6`

	_, err := querier.ExecContext(
		ctx,
		query,
		dek.KekID,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
		dek.ID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update dek")
	}

	return nil
}

// NewPostgreSQLDekRepository creates a new PostgreSQL DEK repository instance.
//
// Parameters:
//   - db: A PostgreSQL database connection
//
// Returns:
//   - A new PostgreSQLDekRepository ready for use
//
// Example:
//
//	db, err := sql.Open("postgres", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewPostgreSQLDekRepository(db)
func NewPostgreSQLDekRepository(db *sql.DB) *PostgreSQLDekRepository {
	return &PostgreSQLDekRepository{db: db}
}
