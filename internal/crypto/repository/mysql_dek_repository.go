package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLDekRepository implements DEK persistence for MySQL databases.
//
// This repository handles storing and retrieving Data Encryption Keys using
// MySQL's BINARY(16) for UUID storage and BLOB for binary data. UUIDs are
// marshaled/unmarshaled to/from binary format using uuid.MarshalBinary() and
// uuid.UnmarshalBinary(). It supports transaction-aware operations via
// database.GetTx(), enabling atomic operations.
//
// Database schema requirements:
//   - id: BINARY(16) PRIMARY KEY (UUID in binary format)
//   - kek_id: BINARY(16) (foreign key reference to KEK)
//   - algorithm: VARCHAR(50) (e.g., "aes-gcm", "chacha20-poly1305")
//   - encrypted_key: BLOB (encrypted DEK bytes)
//   - nonce: BLOB (encryption nonce)
//   - created_at: DATETIME/TIMESTAMP
//
// UUID handling:
//
//	MySQL doesn't have a native UUID type, so UUIDs are stored as BINARY(16).
//	The repository handles marshaling/unmarshaling automatically using
//	uuid.MarshalBinary() and uuid.UnmarshalBinary() methods.
//
// Transaction support:
//
//	The repository automatically detects transaction context using database.GetTx().
//	All methods work both within and outside of transactions seamlessly.
//
// Example usage:
//
//	repo := NewMySQLDekRepository(db)
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
type MySQLDekRepository struct {
	db *sql.DB
}

// Create inserts a new DEK into the MySQL database.
//
// The DEK's ID and KekID are marshaled to BINARY(16) format using uuid.MarshalBinary(),
// and binary fields (EncryptedKey, Nonce) are stored as BLOBs. This method
// supports transaction context via database.GetTx(), enabling atomic multi-step
// operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - dek: The Data Encryption Key to insert (must have all required fields populated)
//
// Returns:
//   - An error if marshaling the UUIDs fails or the insert fails
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
func (m *MySQLDekRepository) Create(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	id, err := dek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	kekID, err := dek.KekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		kekID,
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

// Update modifies an existing DEK in the MySQL database.
//
// This method updates all mutable fields of the DEK. The DEK ID and KekID are
// marshaled to BINARY(16) format using uuid.MarshalBinary(). The method supports
// transaction context via database.GetTx(), enabling atomic operations such as
// re-encrypting a DEK with a new KEK during key rotation.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - dek: The DEK with updated field values (ID must match existing record)
//
// Returns:
//   - An error if marshaling the UUIDs fails or the update fails
//
// Example:
//
//	// Re-encrypt DEK with a new KEK
//	dek.KekID = newKekID
//	dek.EncryptedKey = newEncryptedBytes
//	dek.Nonce = newNonce
//	err := repo.Update(ctx, dek)
func (m *MySQLDekRepository) Update(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE deks 
			  SET kek_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  created_at = ?
			  WHERE id = ?`

	kekID, err := dek.KekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	id, err := dek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		kekID,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update dek")
	}

	return nil
}

// NewMySQLDekRepository creates a new MySQL DEK repository instance.
//
// Parameters:
//   - db: A MySQL database connection
//
// Returns:
//   - A new MySQLDekRepository ready for use
//
// Example:
//
//	db, err := sql.Open("mysql", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewMySQLDekRepository(db)
func NewMySQLDekRepository(db *sql.DB) *MySQLDekRepository {
	return &MySQLDekRepository{db: db}
}
