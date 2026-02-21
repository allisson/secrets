package service

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

func BenchmarkAuditSigner_Sign(b *testing.B) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		b.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.EncryptCapability,
		Path:       "/transit/benchmark",
		Metadata:   map[string]any{"action": "encrypt", "key_version": 1},
		CreatedAt:  time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := signer.Sign(kekKey, log)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuditSigner_Verify(b *testing.B) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		b.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.DecryptCapability,
		Path:       "/transit/benchmark",
		Metadata:   map[string]any{"action": "decrypt", "key_version": 1},
		CreatedAt:  time.Now().UTC(),
	}

	signature, _ := signer.Sign(kekKey, log)
	log.Signature = signature

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := signer.Verify(kekKey, log)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuditSigner_SignWithComplexMetadata(b *testing.B) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		b.Fatal(err)
	}

	// Complex metadata simulating realistic audit log
	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/app/database/credentials",
		Metadata: map[string]any{
			"action":       "update",
			"version":      42,
			"previous_id":  "01933e4a-7890-7abc-def0-123456789abc",
			"tags":         []string{"prod", "database", "critical"},
			"size_bytes":   1024,
			"encrypted_by": "KEK-v3",
		},
		CreatedAt: time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := signer.Sign(kekKey, log)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuditSigner_BatchSign(b *testing.B) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		b.Fatal(err)
	}

	// Pre-generate batch of logs
	batchSize := 1000
	logs := make([]*authDomain.AuditLog, batchSize)
	for i := 0; i < batchSize; i++ {
		logs[i] = &authDomain.AuditLog{
			ID:         uuid.Must(uuid.NewV7()),
			RequestID:  uuid.Must(uuid.NewV7()),
			ClientID:   uuid.Must(uuid.NewV7()),
			Capability: authDomain.ReadCapability,
			Path:       "/secrets/test",
			Metadata:   map[string]any{"index": i},
			CreatedAt:  time.Now().UTC(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, log := range logs {
			_, err := signer.Sign(kekKey, log)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
