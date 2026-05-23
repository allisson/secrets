package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/keyring"
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

func (t *transitKeyUseCaseWithMetrics) Create(
	ctx context.Context,
	name string,
	alg keyring.Algorithm,
) (*transitDomain.TransitKey, error) {
	start := time.Now()
	key, err := t.next.Create(ctx, name, alg)
	metrics.Record(ctx, t.metrics, "transit", "transit_key_create", start, err)
	return key, err
}

func (t *transitKeyUseCaseWithMetrics) Rotate(
	ctx context.Context,
	name string,
	alg keyring.Algorithm,
) (*transitDomain.TransitKey, error) {
	start := time.Now()
	key, err := t.next.Rotate(ctx, name, alg)
	metrics.Record(ctx, t.metrics, "transit", "transit_key_rotate", start, err)
	return key, err
}

func (t *transitKeyUseCaseWithMetrics) Get(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, keyring.Algorithm, error) {
	start := time.Now()
	key, alg, err := t.next.Get(ctx, name, version)
	metrics.Record(ctx, t.metrics, "transit", "transit_key_get", start, err)
	return key, alg, err
}

func (t *transitKeyUseCaseWithMetrics) Delete(ctx context.Context, name string) error {
	start := time.Now()
	err := t.next.Delete(ctx, name)
	metrics.Record(ctx, t.metrics, "transit", "transit_key_delete", start, err)
	return err
}

func (t *transitKeyUseCaseWithMetrics) Encrypt(
	ctx context.Context,
	name string,
	plaintext, context []byte,
) (*transitDomain.EncryptedBlob, error) {
	start := time.Now()
	blob, err := t.next.Encrypt(ctx, name, plaintext, context)
	metrics.Record(ctx, t.metrics, "transit", "transit_encrypt", start, err)
	return blob, err
}

func (t *transitKeyUseCaseWithMetrics) Decrypt(
	ctx context.Context,
	name string,
	ciphertext string,
	context []byte,
) (*transitDomain.EncryptedBlob, error) {
	start := time.Now()
	blob, err := t.next.Decrypt(ctx, name, ciphertext, context)
	metrics.Record(ctx, t.metrics, "transit", "transit_decrypt", start, err)
	return blob, err
}

func (t *transitKeyUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*transitDomain.TransitKey, error) {
	start := time.Now()
	keys, err := t.next.ListCursor(ctx, afterName, limit)
	metrics.Record(ctx, t.metrics, "transit", "transit_key_list", start, err)
	return keys, err
}

func (t *transitKeyUseCaseWithMetrics) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (int64, error) {
	start := time.Now()
	count, err := t.next.PurgeDeleted(ctx, olderThanDays, dryRun)
	metrics.Record(ctx, t.metrics, "transit", "transit_key_purge", start, err)
	return count, err
}
