package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/metrics"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// transitKeyUseCaseWithMetrics decorates TransitKeyUseCase with metrics instrumentation.
type transitKeyUseCaseWithMetrics struct {
	next    TransitKeyUseCase
	metrics metrics.BusinessMetrics
}

// NewTransitKeyUseCaseWithMetrics wraps a TransitKeyUseCase with metrics recording.
func NewTransitKeyUseCaseWithMetrics(useCase TransitKeyUseCase, m metrics.BusinessMetrics) TransitKeyUseCase {
	return &transitKeyUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

// Create records metrics for transit key creation operations.
func (t *transitKeyUseCaseWithMetrics) Create(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
	start := time.Now()
	key, err := t.next.Create(ctx, name, alg)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "transit", "transit_key_create", status)
	t.metrics.RecordDuration(ctx, "transit", "transit_key_create", time.Since(start), status)

	return key, err
}

// Rotate records metrics for transit key rotation operations.
func (t *transitKeyUseCaseWithMetrics) Rotate(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
	start := time.Now()
	key, err := t.next.Rotate(ctx, name, alg)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "transit", "transit_key_rotate", status)
	t.metrics.RecordDuration(ctx, "transit", "transit_key_rotate", time.Since(start), status)

	return key, err
}

// Delete records metrics for transit key deletion operations.
func (t *transitKeyUseCaseWithMetrics) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	start := time.Now()
	err := t.next.Delete(ctx, transitKeyID)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "transit", "transit_key_delete", status)
	t.metrics.RecordDuration(ctx, "transit", "transit_key_delete", time.Since(start), status)

	return err
}

// Encrypt records metrics for transit encryption operations.
func (t *transitKeyUseCaseWithMetrics) Encrypt(
	ctx context.Context,
	name string,
	plaintext []byte,
) (*transitDomain.EncryptedBlob, error) {
	start := time.Now()
	blob, err := t.next.Encrypt(ctx, name, plaintext)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "transit", "transit_encrypt", status)
	t.metrics.RecordDuration(ctx, "transit", "transit_encrypt", time.Since(start), status)

	return blob, err
}

// Decrypt records metrics for transit decryption operations.
func (t *transitKeyUseCaseWithMetrics) Decrypt(
	ctx context.Context,
	name string,
	ciphertext string,
) (*transitDomain.EncryptedBlob, error) {
	start := time.Now()
	blob, err := t.next.Decrypt(ctx, name, ciphertext)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "transit", "transit_decrypt", status)
	t.metrics.RecordDuration(ctx, "transit", "transit_decrypt", time.Since(start), status)

	return blob, err
}
