package keyring

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type keyManager interface {
	createKek(masterKey *MasterKey, alg Algorithm) (kek, error)
	decryptKek(k *kek, masterKey *MasterKey) ([]byte, error)
	createDek(k *kek, alg Algorithm) (dek, error)
	encryptDek(dekKey []byte, k *kek) (ciphertext, nonce []byte, err error)
	decryptDek(d *dek, k *kek) ([]byte, error)
}

type keyManagerService struct {
	aeadManager aeadManager
}

func newKeyManager(am aeadManager) keyManager {
	return &keyManagerService{aeadManager: am}
}

func (km *keyManagerService) createKek(masterKey *MasterKey, alg Algorithm) (kek, error) {
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		return kek{}, fmt.Errorf("failed to generate KEK: %w", err)
	}
	defer Zero(kekKey)

	cipher, err := km.aeadManager.createCipher(masterKey.Key, alg)
	if err != nil {
		return kek{}, err
	}

	encryptedKey, nonce, err := cipher.Encrypt(kekKey, nil)
	if err != nil {
		return kek{}, fmt.Errorf("failed to encrypt KEK: %w", err)
	}

	keyCopy := make([]byte, len(kekKey))
	copy(keyCopy, kekKey)

	k := kek{
		id:           uuid.Must(uuid.NewV7()),
		masterKeyID:  masterKey.ID,
		algorithm:    alg,
		encryptedKey: encryptedKey,
		key:          keyCopy,
		nonce:        nonce,
		version:      1,
		createdAt:    time.Now().UTC(),
	}

	return k, nil
}

func (km *keyManagerService) decryptKek(k *kek, masterKey *MasterKey) ([]byte, error) {
	cipher, err := km.aeadManager.createCipher(masterKey.Key, k.algorithm)
	if err != nil {
		return nil, err
	}

	kekKey, err := cipher.Decrypt(k.encryptedKey, k.nonce, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return kekKey, nil
}

func (km *keyManagerService) createDek(k *kek, alg Algorithm) (dek, error) {
	dekKey := make([]byte, 32)
	if _, err := rand.Read(dekKey); err != nil {
		return dek{}, fmt.Errorf("failed to generate DEK: %w", err)
	}
	defer Zero(dekKey)

	cipher, err := km.aeadManager.createCipher(k.key, k.algorithm)
	if err != nil {
		return dek{}, err
	}

	encryptedKey, nonce, err := cipher.Encrypt(dekKey, nil)
	if err != nil {
		return dek{}, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	d := dek{
		id:           uuid.Must(uuid.NewV7()),
		kekID:        k.id,
		algorithm:    alg,
		encryptedKey: encryptedKey,
		nonce:        nonce,
		createdAt:    time.Now().UTC(),
	}

	return d, nil
}

func (km *keyManagerService) encryptDek(dekKey []byte, k *kek) (ciphertext, nonce []byte, err error) {
	cipher, err := km.aeadManager.createCipher(k.key, k.algorithm)
	if err != nil {
		return nil, nil, err
	}

	encryptedKey, n, err := cipher.Encrypt(dekKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	return encryptedKey, n, nil
}

func (km *keyManagerService) decryptDek(d *dek, k *kek) ([]byte, error) {
	cipher, err := km.aeadManager.createCipher(k.key, k.algorithm)
	if err != nil {
		return nil, err
	}

	dekKey, err := cipher.Decrypt(d.encryptedKey, d.nonce, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return dekKey, nil
}
