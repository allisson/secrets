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
)

var errKeyringBadBatchSize = errors.New("keyring: batch size must be positive")

// keyringImpl is the production Keyring. It orchestrates KEK chain lookup, DEK
// creation/persistence, AEAD cipher construction, and ciphertext I/O.
type keyringImpl struct {
	kekChain     *kekChain
	dekStore     dekStore
	aeadManager  aeadManager
	keyManager   keyManager
	dekAlgorithm Algorithm
}

func (k *keyringImpl) Encrypt(ctx context.Context, plaintext []byte) (Envelope, error) {
	kk, err := k.activeKek()
	if err != nil {
		return Envelope{}, err
	}

	d, err := k.createAndPersistDek(ctx, kk, k.dekAlgorithm)
	if err != nil {
		return Envelope{}, err
	}

	dekKey, err := k.keyManager.decryptDek(&d, kk)
	if err != nil {
		return Envelope{}, err
	}
	defer Zero(dekKey)

	cipher, err := k.aeadManager.createCipher(dekKey, k.dekAlgorithm)
	if err != nil {
		return Envelope{}, err
	}

	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		DekID:      d.id,
		Ciphertext: ciphertext,
		Nonce:      nonce,
	}, nil
}

func (k *keyringImpl) Decrypt(ctx context.Context, env Envelope) ([]byte, error) {
	d, err := k.dekStore.get(ctx, env.DekID)
	if err != nil {
		if errors.Is(err, ErrDekNotFound) {
			return nil, ErrDecryptionFailed
		}
		return nil, err
	}

	kk, ok := k.kekChain.get(d.kekID)
	if !ok {
		return nil, ErrDecryptionFailed
	}

	dekKey, err := k.keyManager.decryptDek(d, kk)
	if err != nil {
		return nil, err
	}
	defer Zero(dekKey)

	cipher, err := k.aeadManager.createCipher(dekKey, d.algorithm)
	if err != nil {
		return nil, err
	}

	return cipher.Decrypt(env.Ciphertext, env.Nonce, nil)
}

func (k *keyringImpl) AllocateDek(ctx context.Context, alg Algorithm) (DekHandle, error) {
	kk, err := k.activeKek()
	if err != nil {
		return DekHandle{}, err
	}

	d, err := k.createAndPersistDek(ctx, kk, alg)
	if err != nil {
		return DekHandle{}, err
	}

	return DekHandle{DekID: d.id}, nil
}

func (k *keyringImpl) EncryptWith(
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

func (k *keyringImpl) DecryptWith(
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

func (k *keyringImpl) Rewrap(ctx context.Context, dekID uuid.UUID) error {
	d, err := k.dekStore.get(ctx, dekID)
	if err != nil {
		return err
	}

	activeKek, err := k.activeKek()
	if err != nil {
		return err
	}

	if d.kekID == activeKek.id {
		return nil
	}

	oldKek, ok := k.kekChain.get(d.kekID)
	if !ok {
		return ErrKekNotFound
	}

	dekKey, err := k.keyManager.decryptDek(d, oldKek)
	if err != nil {
		return err
	}
	defer Zero(dekKey)

	newEncKey, newNonce, err := k.keyManager.encryptDek(dekKey, activeKek)
	if err != nil {
		return err
	}

	d.kekID = activeKek.id
	d.encryptedKey = newEncKey
	d.nonce = newNonce

	return k.dekStore.update(ctx, d)
}

func (k *keyringImpl) RewrapAll(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		return 0, errKeyringBadBatchSize
	}

	activeKekID := k.kekChain.activeKekID()
	total := 0
	for {
		deks, err := k.dekStore.getBatchNotKekID(ctx, activeKekID, batchSize)
		if err != nil {
			return total, err
		}
		if len(deks) == 0 {
			return total, nil
		}
		for _, d := range deks {
			if err := k.Rewrap(ctx, d.id); err != nil {
				return total, err
			}
			total++
		}
	}
}

func (k *keyringImpl) ActiveKekID() uuid.UUID {
	return k.kekChain.activeKekID()
}

func (k *keyringImpl) SignWithKey(data []byte) ([]byte, uuid.UUID, error) {
	kk, err := k.activeKek()
	if err != nil {
		return nil, uuid.Nil, err
	}

	signingKey, err := deriveAuditSigningKey(kk.key)
	if err != nil {
		return nil, uuid.Nil, err
	}
	defer Zero(signingKey)

	mac := hmac.New(sha256.New, signingKey)
	mac.Write(data)
	return mac.Sum(nil), kk.id, nil
}

func (k *keyringImpl) VerifyWithKey(kekID uuid.UUID, data, sig []byte) error {
	kk, ok := k.kekChain.get(kekID)
	if !ok {
		return ErrKekNotFound
	}

	signingKey, err := deriveAuditSigningKey(kk.key)
	if err != nil {
		return err
	}
	defer Zero(signingKey)

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

func (k *keyringImpl) activeKek() (*kek, error) {
	kk, ok := k.kekChain.get(k.kekChain.activeKekID())
	if !ok {
		return nil, ErrKekNotFound
	}
	return kk, nil
}

func (k *keyringImpl) createAndPersistDek(
	ctx context.Context,
	kk *kek,
	alg Algorithm,
) (dek, error) {
	d, err := k.keyManager.createDek(kk, alg)
	if err != nil {
		return dek{}, err
	}

	if d.createdAt.IsZero() {
		d.createdAt = time.Now().UTC()
	}

	if err := k.dekStore.create(ctx, &d); err != nil {
		return dek{}, err
	}

	return d, nil
}

// openCipher loads the DEK referenced by handle, unwraps it under its KEK,
// and returns an AEAD cipher plus a cleanup function that zeroes the DEK key.
func (k *keyringImpl) openCipher(
	ctx context.Context,
	handle DekHandle,
) (aead, func(), error) {
	d, err := k.dekStore.get(ctx, handle.DekID)
	if err != nil {
		if errors.Is(err, ErrDekNotFound) {
			return nil, func() {}, ErrDecryptionFailed
		}
		return nil, func() {}, err
	}

	kk, ok := k.kekChain.get(d.kekID)
	if !ok {
		return nil, func() {}, ErrDecryptionFailed
	}

	dekKey, err := k.keyManager.decryptDek(d, kk)
	if err != nil {
		return nil, func() {}, err
	}

	cipher, err := k.aeadManager.createCipher(dekKey, d.algorithm)
	if err != nil {
		Zero(dekKey)
		return nil, func() {}, err
	}

	cleanup := func() { Zero(dekKey) }
	return cipher, cleanup, nil
}
