package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/keyring"
	"github.com/allisson/secrets/internal/metrics"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

type metricsTransitKeyUseCase struct {
	inner  TransitKeyUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsTransitKeyUseCase wraps inner with per-method timing and operation recording.
func NewMetricsTransitKeyUseCase(
	inner TransitKeyUseCase,
	bm metrics.BusinessMetrics,
	domain string,
) TransitKeyUseCase {
	return &metricsTransitKeyUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsTransitKeyUseCase) Create(
	ctx context.Context,
	name string,
	alg keyring.Algorithm,
) (result *transitDomain.TransitKey, err error) {
	start := time.Now()
	result, err = m.inner.Create(ctx, name, alg)
	metrics.Record(ctx, m.bm, m.domain, "transit_key_create", start, err)
	return
}

func (m *metricsTransitKeyUseCase) Rotate(
	ctx context.Context,
	name string,
	alg keyring.Algorithm,
) (result *transitDomain.TransitKey, err error) {
	start := time.Now()
	result, err = m.inner.Rotate(ctx, name, alg)
	metrics.Record(ctx, m.bm, m.domain, "transit_key_rotate", start, err)
	return
}

func (m *metricsTransitKeyUseCase) Get(
	ctx context.Context,
	name string,
	version uint,
) (key *transitDomain.TransitKey, err error) {
	start := time.Now()
	key, err = m.inner.Get(ctx, name, version)
	metrics.Record(ctx, m.bm, m.domain, "transit_key_get", start, err)
	return
}

func (m *metricsTransitKeyUseCase) Delete(ctx context.Context, name string) (err error) {
	start := time.Now()
	err = m.inner.Delete(ctx, name)
	metrics.Record(ctx, m.bm, m.domain, "transit_key_delete", start, err)
	return
}

func (m *metricsTransitKeyUseCase) Encrypt(
	ctx context.Context,
	name string,
	plaintext, aad []byte,
) (result *transitDomain.EncryptedBlob, err error) {
	start := time.Now()
	result, err = m.inner.Encrypt(ctx, name, plaintext, aad)
	metrics.Record(ctx, m.bm, m.domain, "transit_encrypt", start, err)
	return
}

func (m *metricsTransitKeyUseCase) Decrypt(
	ctx context.Context,
	name string,
	ciphertext string,
	aad []byte,
) (result *transitDomain.EncryptedBlob, err error) {
	start := time.Now()
	result, err = m.inner.Decrypt(ctx, name, ciphertext, aad)
	metrics.Record(ctx, m.bm, m.domain, "transit_decrypt", start, err)
	return
}

func (m *metricsTransitKeyUseCase) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) (result []*transitDomain.TransitKey, err error) {
	start := time.Now()
	result, err = m.inner.ListCursor(ctx, afterName, limit)
	metrics.Record(ctx, m.bm, m.domain, "transit_key_list", start, err)
	return
}

func (m *metricsTransitKeyUseCase) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (count int64, err error) {
	start := time.Now()
	count, err = m.inner.PurgeDeleted(ctx, olderThanDays, dryRun)
	metrics.Record(ctx, m.bm, m.domain, "transit_key_purge_deleted", start, err)
	return
}

var _ TransitKeyUseCase = (*metricsTransitKeyUseCase)(nil)
