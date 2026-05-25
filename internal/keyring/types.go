package keyring

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type kek struct {
	id           uuid.UUID
	masterKeyID  string
	algorithm    Algorithm
	encryptedKey []byte
	key          []byte
	nonce        []byte
	version      uint
	createdAt    time.Time
}

type kekChain struct {
	activeID uuid.UUID
	keys     sync.Map
}

func (k *kekChain) activeKekID() uuid.UUID {
	return k.activeID
}

func (k *kekChain) get(id uuid.UUID) (*kek, bool) {
	if v, ok := k.keys.Load(id); ok {
		return v.(*kek), true
	}
	return nil, false
}

func newKekChain(keks []*kek) *kekChain {
	if len(keks) == 0 {
		return &kekChain{}
	}

	kc := &kekChain{
		activeID: keks[0].id,
	}

	for _, k := range keks {
		kc.keys.Store(k.id, k)
	}

	return kc
}

type dek struct {
	id           uuid.UUID
	kekID        uuid.UUID
	algorithm    Algorithm
	encryptedKey []byte
	nonce        []byte
	createdAt    time.Time
}
