package keyring

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/hkdf"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
)

var errKeyringBadBatchSize = errors.New("keyring: batch size must be positive")

// dekStore is the persistence contract Keyring relies on for DEK rows.
//
// Implemented by *crypto/repository.DekRepository today. Kept narrow here so
// keyring can swap to its own repository once internal/crypto is folded in.
type dekStore interface {
	Create(ctx context.Context, dek *cryptoDomain.Dek) error
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
	Update(ctx context.Context, dek *cryptoDomain.Dek) error
	GetBatchNotKekID(ctx context.Context, kekID uuid.UUID, limit int) ([]*cryptoDomain.Dek, error)
}

// keyring is the production Keyring. It orchestrates KEK chain lookup, DEK
// creation/persistence, AEAD cipher construction, and ciphertext I/O.
type keyring struct {
	kekChain     *cryptoDomain.KekChain
	dekStore     dekStore
	aeadManager  cryptoService.AEADManager
	keyManager   cryptoService.KeyManager
	dekAlgorithm Algorithm
}

// New constructs a Keyring with the given dependencies. The kekChain must be
// non-empty and have at least one usable KEK (the active one).
func New(
	kekChain *cryptoDomain.KekChain,
	dekStore dekStore,
	aeadManager cryptoService.AEADManager,
	keyManager cryptoService.KeyManager,
	dekAlgorithm Algorithm,
) Keyring {
	return &keyring{
		kekChain:     kekChain,
		dekStore:     dekStore,
		aeadManager:  aeadManager,
		keyManager:   keyManager,
		dekAlgorithm: dekAlgorithm,
	}
}

func (k *keyring) Encrypt(ctx context.Context, plaintext []byte) (Envelope, error) {
	kek, err := k.activeKek()
	if err != nil {
		return Envelope{}, err
	}

	dek, err := k.createAndPersistDek(ctx, kek, k.dekAlgorithm)
	if err != nil {
		return Envelope{}, err
	}

	dekKey, err := k.keyManager.DecryptDek(&dek, kek)
	if err != nil {
		return Envelope{}, err
	}
	defer cryptoDomain.Zero(dekKey)

	cipher, err := k.aeadManager.CreateCipher(dekKey, k.dekAlgorithm)
	if err != nil {
		return Envelope{}, err
	}

	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		DekID:      dek.ID,
		Ciphertext: ciphertext,
		Nonce:      nonce,
	}, nil
}

func (k *keyring) Decrypt(ctx context.Context, env Envelope) ([]byte, error) {
	dek, err := k.dekStore.Get(ctx, env.DekID)
	if err != nil {
		if errors.Is(err, cryptoDomain.ErrDekNotFound) {
			return nil, ErrDecryptionFailed
		}
		return nil, err
	}

	kek, ok := k.kekChain.Get(dek.KekID)
	if !ok {
		return nil, ErrDecryptionFailed
	}

	dekKey, err := k.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}
	defer cryptoDomain.Zero(dekKey)

	cipher, err := k.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	return cipher.Decrypt(env.Ciphertext, env.Nonce, nil)
}

func (k *keyring) AllocateDek(ctx context.Context, alg Algorithm) (DekHandle, error) {
	kek, err := k.activeKek()
	if err != nil {
		return DekHandle{}, err
	}

	dek, err := k.createAndPersistDek(ctx, kek, alg)
	if err != nil {
		return DekHandle{}, err
	}

	return DekHandle{DekID: dek.ID}, nil
}

func (k *keyring) EncryptWith(
	ctx context.Context,
	handle DekHandle,
	plaintext, aad []byte,
) (ciphertext, nonce []byte, err error) {
	cipher, cleanup, err := k.openCipher(ctx, handle)
	if err != nil {
		return nil, nil, err
	}
	defer cleanup()

	return cipher.Encrypt(plaintext, aad)
}

func (k *keyring) DecryptWith(
	ctx context.Context,
	handle DekHandle,
	ciphertext, nonce, aad []byte,
) ([]byte, error) {
	cipher, cleanup, err := k.openCipher(ctx, handle)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return cipher.Decrypt(ciphertext, nonce, aad)
}

func (k *keyring) Rewrap(ctx context.Context, dekID uuid.UUID) error {
	dek, err := k.dekStore.Get(ctx, dekID)
	if err != nil {
		return err
	}

	activeKek, err := k.activeKek()
	if err != nil {
		return err
	}

	if dek.KekID == activeKek.ID {
		return nil
	}

	oldKek, ok := k.kekChain.Get(dek.KekID)
	if !ok {
		return cryptoDomain.ErrKekNotFound
	}

	dekKey, err := k.keyManager.DecryptDek(dek, oldKek)
	if err != nil {
		return err
	}
	defer cryptoDomain.Zero(dekKey)

	newEncKey, newNonce, err := k.keyManager.EncryptDek(dekKey, activeKek)
	if err != nil {
		return err
	}

	dek.KekID = activeKek.ID
	dek.EncryptedKey = newEncKey
	dek.Nonce = newNonce

	return k.dekStore.Update(ctx, dek)
}

func (k *keyring) RewrapAll(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		return 0, errKeyringBadBatchSize
	}

	activeKekID := k.kekChain.ActiveKekID()
	total := 0
	for {
		deks, err := k.dekStore.GetBatchNotKekID(ctx, activeKekID, batchSize)
		if err != nil {
			return total, err
		}
		if len(deks) == 0 {
			return total, nil
		}
		for _, dek := range deks {
			if err := k.Rewrap(ctx, dek.ID); err != nil {
				return total, err
			}
			total++
		}
	}
}

func (k *keyring) ActiveKekID() uuid.UUID {
	return k.kekChain.ActiveKekID()
}

func (k *keyring) SignWithKey(data []byte) ([]byte, uuid.UUID, error) {
	kek, err := k.activeKek()
	if err != nil {
		return nil, uuid.Nil, err
	}

	signingKey, err := deriveAuditSigningKey(kek.Key)
	if err != nil {
		return nil, uuid.Nil, err
	}
	defer cryptoDomain.Zero(signingKey)

	mac := hmac.New(sha256.New, signingKey)
	mac.Write(data)
	return mac.Sum(nil), kek.ID, nil
}

func (k *keyring) VerifyWithKey(kekID uuid.UUID, data, sig []byte) error {
	kek, ok := k.kekChain.Get(kekID)
	if !ok {
		return ErrKekNotFound
	}

	signingKey, err := deriveAuditSigningKey(kek.Key)
	if err != nil {
		return err
	}
	defer cryptoDomain.Zero(signingKey)

	mac := hmac.New(sha256.New, signingKey)
	mac.Write(data)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return ErrSignatureInvalid
	}
	return nil
}

// deriveAuditSigningKey derives a 32-byte HMAC signing key from a KEK via HKDF-SHA256.
// The "audit-log-signing-v1" info parameter separates signing key usage from encryption.
func deriveAuditSigningKey(kekKey []byte) ([]byte, error) {
	reader := hkdf.New(sha256.New, kekKey, nil, []byte("audit-log-signing-v1"))
	signingKey := make([]byte, 32)
	if _, err := io.ReadFull(reader, signingKey); err != nil {
		return nil, err
	}
	return signingKey, nil
}

func (k *keyring) activeKek() (*cryptoDomain.Kek, error) {
	kek, ok := k.kekChain.Get(k.kekChain.ActiveKekID())
	if !ok {
		return nil, cryptoDomain.ErrKekNotFound
	}
	return kek, nil
}

func (k *keyring) createAndPersistDek(
	ctx context.Context,
	kek *cryptoDomain.Kek,
	alg Algorithm,
) (cryptoDomain.Dek, error) {
	dek, err := k.keyManager.CreateDek(kek, alg)
	if err != nil {
		return cryptoDomain.Dek{}, err
	}

	if dek.CreatedAt.IsZero() {
		dek.CreatedAt = time.Now().UTC()
	}

	if err := k.dekStore.Create(ctx, &dek); err != nil {
		return cryptoDomain.Dek{}, err
	}

	return dek, nil
}

// openCipher loads the DEK referenced by handle, unwraps it under its KEK,
// and returns an AEAD cipher plus a cleanup function that zeroes the DEK.
func (k *keyring) openCipher(
	ctx context.Context,
	handle DekHandle,
) (cryptoService.AEAD, func(), error) {
	dek, err := k.dekStore.Get(ctx, handle.DekID)
	if err != nil {
		if errors.Is(err, cryptoDomain.ErrDekNotFound) {
			return nil, func() {}, ErrDecryptionFailed
		}
		return nil, func() {}, err
	}

	kek, ok := k.kekChain.Get(dek.KekID)
	if !ok {
		return nil, func() {}, ErrDecryptionFailed
	}

	dekKey, err := k.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, func() {}, err
	}

	cipher, err := k.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		cryptoDomain.Zero(dekKey)
		return nil, func() {}, err
	}

	cleanup := func() { cryptoDomain.Zero(dekKey) }
	return cipher, cleanup, nil
}
