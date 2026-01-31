package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLKekRepository implements KEK persistence for MySQL databases.
//
// This repository handles storing and retrieving Key Encryption Keys using
// MySQL's BINARY(16) for UUID storage and BLOB for binary data. UUIDs are
// marshaled/unmarshaled to/from binary format using uuid.MarshalBinary() and
// uuid.UnmarshalBinary(). It supports transaction-aware operations via
// database.GetTx(), enabling atomic key rotation operations.
//
// Database schema requirements:
//   - id: BINARY(16) PRIMARY KEY (UUID in binary format)
//   - master_key_id: VARCHAR(255) (reference to master key)
//   - algorithm: VARCHAR(50) (e.g., "aes-gcm", "chacha20-poly1305")
//   - encrypted_key: BLOB (encrypted KEK bytes)
//   - nonce: BLOB (encryption nonce)
//   - version: INT UNSIGNED (for tracking KEK versions during rotation)
//   - is_active: BOOLEAN/TINYINT(1) (indicates the active KEK)
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
//	repo := NewMySQLKekRepository(db)
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
type MySQLKekRepository struct {
	db *sql.DB
}

// Create inserts a new KEK into the MySQL database.
//
// The KEK's ID is marshaled to BINARY(16) format using uuid.MarshalBinary(),
// and binary fields (EncryptedKey, Nonce) are stored as BLOBs. This method
// supports transaction context via database.GetTx(), enabling atomic multi-step
// operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - kek: The Key Encryption Key to insert (must have all required fields populated)
//
// Returns:
//   - An error if marshaling the UUID fails or the insert fails
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
//	    IsActive:     true,
//	    CreatedAt:    time.Now().UTC(),
//	}
//	err := repo.Create(ctx, kek)
func (m *MySQLKekRepository) Create(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, is_active, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	id, err := kek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.IsActive,
		kek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create kek")
	}
	return nil
}

// Update modifies an existing KEK in the MySQL database.
//
// This method updates all mutable fields of the KEK. The KEK ID is marshaled
// to BINARY(16) format for the WHERE clause using uuid.MarshalBinary(). It's
// typically used to deactivate old KEKs during key rotation by setting IsActive
// to false. The method supports transaction context via database.GetTx(),
// enabling atomic rotation operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - kek: The KEK with updated field values (ID must match existing record)
//
// Returns:
//   - An error if marshaling the UUID fails or the update fails
//
// Example:
//
//	// Deactivate old KEK during rotation
//	oldKek.IsActive = false
//	err := repo.Update(ctx, oldKek)
func (m *MySQLKekRepository) Update(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE keks 
			  SET master_key_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  version = ?, 
			      is_active = ?,
				  created_at = ?
			  WHERE id = ?`

	id, err := kek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.IsActive,
		kek.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update kek")
	}

	return nil
}

// List retrieves all KEKs from the MySQL database ordered by version descending.
//
// This method returns all KEKs (both active and inactive) sorted by version
// in descending order. KEK IDs are automatically unmarshaled from BINARY(16)
// to uuid.UUID using uuid.UnmarshalBinary(). This ordering is critical for
// key rotation scenarios where you need to identify the latest KEK version
// to set as active in the KekChain.
//
// The method supports transaction context via database.GetTx(), allowing
// consistent reads within a transaction.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//
// Returns:
//   - A slice of KEK pointers ordered by version descending (newest first)
//   - An error if the query fails or UUID unmarshaling fails
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
func (m *MySQLKekRepository) List(ctx context.Context) ([]*cryptoDomain.Kek, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, master_key_id, algorithm, encrypted_key, nonce, version, is_active, created_at 
			  FROM keks ORDER BY version DESC`

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
		var id []byte

		err := rows.Scan(
			&id,
			&kek.MasterKeyID,
			&kek.Algorithm,
			&kek.EncryptedKey,
			&kek.Nonce,
			&kek.Version,
			&kek.IsActive,
			&kek.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := kek.ID.UnmarshalBinary(id); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal kek id")
		}

		keks = append(keks, &kek)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keks, nil
}

// NewMySQLKekRepository creates a new MySQL KEK repository instance.
//
// Parameters:
//   - db: A MySQL database connection
//
// Returns:
//   - A new MySQLKekRepository ready for use
//
// Example:
//
//	db, err := sql.Open("mysql", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewMySQLKekRepository(db)
func NewMySQLKekRepository(db *sql.DB) *MySQLKekRepository {
	return &MySQLKekRepository{db: db}
}
