package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/metrics"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

type metricsSecretUseCase struct {
	inner  SecretUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsSecretUseCase wraps inner with per-method timing and operation recording.
func NewMetricsSecretUseCase(inner SecretUseCase, bm metrics.BusinessMetrics, domain string) SecretUseCase {
	return &metricsSecretUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsSecretUseCase) CreateOrUpdate(
	ctx context.Context,
	path string,
	value []byte,
) (result *secretsDomain.Secret, err error) {
	start := time.Now()
	result, err = m.inner.CreateOrUpdate(ctx, path, value)
	metrics.Record(ctx, m.bm, m.domain, "secret_create_or_update", start, err)
	return
}

func (m *metricsSecretUseCase) Get(
	ctx context.Context,
	path string,
) (result *secretsDomain.Secret, err error) {
	start := time.Now()
	result, err = m.inner.Get(ctx, path)
	metrics.Record(ctx, m.bm, m.domain, "secret_get", start, err)
	return
}

func (m *metricsSecretUseCase) GetByVersion(
	ctx context.Context,
	path string,
	version uint,
) (result *secretsDomain.Secret, err error) {
	start := time.Now()
	result, err = m.inner.GetByVersion(ctx, path, version)
	metrics.Record(ctx, m.bm, m.domain, "secret_get_by_version", start, err)
	return
}

func (m *metricsSecretUseCase) Delete(ctx context.Context, path string) (err error) {
	start := time.Now()
	err = m.inner.Delete(ctx, path)
	metrics.Record(ctx, m.bm, m.domain, "secret_delete", start, err)
	return
}

func (m *metricsSecretUseCase) ListCursor(
	ctx context.Context,
	afterPath *string,
	limit int,
) (result []*secretsDomain.Secret, err error) {
	start := time.Now()
	result, err = m.inner.ListCursor(ctx, afterPath, limit)
	metrics.Record(ctx, m.bm, m.domain, "secret_list", start, err)
	return
}

func (m *metricsSecretUseCase) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (count int64, err error) {
	start := time.Now()
	count, err = m.inner.PurgeDeleted(ctx, olderThanDays, dryRun)
	metrics.Record(ctx, m.bm, m.domain, "secret_purge_deleted", start, err)
	return
}

var _ SecretUseCase = (*metricsSecretUseCase)(nil)
