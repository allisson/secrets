package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/keyring"
	"github.com/allisson/secrets/internal/metrics"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

type metricsTokenizationKeyUseCase struct {
	inner  TokenizationKeyUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsTokenizationKeyUseCase wraps inner with per-method timing and operation recording.
func NewMetricsTokenizationKeyUseCase(
	inner TokenizationKeyUseCase,
	bm metrics.BusinessMetrics,
	domain string,
) TokenizationKeyUseCase {
	return &metricsTokenizationKeyUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsTokenizationKeyUseCase) Create(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg keyring.Algorithm,
) (result *tokenizationDomain.TokenizationKey, err error) {
	start := time.Now()
	result, err = m.inner.Create(ctx, name, formatType, isDeterministic, alg)
	metrics.Record(ctx, m.bm, m.domain, "tokenization_key_create", start, err)
	return
}

func (m *metricsTokenizationKeyUseCase) Rotate(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg keyring.Algorithm,
) (result *tokenizationDomain.TokenizationKey, err error) {
	start := time.Now()
	result, err = m.inner.Rotate(ctx, name, formatType, isDeterministic, alg)
	metrics.Record(ctx, m.bm, m.domain, "tokenization_key_rotate", start, err)
	return
}

func (m *metricsTokenizationKeyUseCase) Delete(ctx context.Context, name string) (err error) {
	start := time.Now()
	err = m.inner.Delete(ctx, name)
	metrics.Record(ctx, m.bm, m.domain, "tokenization_key_delete", start, err)
	return
}

func (m *metricsTokenizationKeyUseCase) GetByName(
	ctx context.Context,
	name string,
) (result *tokenizationDomain.TokenizationKey, err error) {
	start := time.Now()
	result, err = m.inner.GetByName(ctx, name)
	metrics.Record(ctx, m.bm, m.domain, "tokenization_key_get", start, err)
	return
}

func (m *metricsTokenizationKeyUseCase) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) (result []*tokenizationDomain.TokenizationKey, err error) {
	start := time.Now()
	result, err = m.inner.ListCursor(ctx, afterName, limit)
	metrics.Record(ctx, m.bm, m.domain, "tokenization_key_list", start, err)
	return
}

func (m *metricsTokenizationKeyUseCase) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (count int64, err error) {
	start := time.Now()
	count, err = m.inner.PurgeDeleted(ctx, olderThanDays, dryRun)
	metrics.Record(ctx, m.bm, m.domain, "tokenization_key_purge_deleted", start, err)
	return
}

var _ TokenizationKeyUseCase = (*metricsTokenizationKeyUseCase)(nil)
