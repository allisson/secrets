package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/metrics"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

type metricsTokenizationUseCase struct {
	inner  TokenizationUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsTokenizationUseCase wraps inner with per-method timing and operation recording.
func NewMetricsTokenizationUseCase(
	inner TokenizationUseCase,
	bm metrics.BusinessMetrics,
	domain string,
) TokenizationUseCase {
	return &metricsTokenizationUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsTokenizationUseCase) Tokenize(
	ctx context.Context,
	keyName string,
	plaintext []byte,
	metadata map[string]any,
	expiresAt *time.Time,
) (result *tokenizationDomain.Token, err error) {
	start := time.Now()
	result, err = m.inner.Tokenize(ctx, keyName, plaintext, metadata, expiresAt)
	metrics.Record(ctx, m.bm, m.domain, "tokenize", start, err)
	return
}

func (m *metricsTokenizationUseCase) TokenizeBatch(
	ctx context.Context,
	keyName string,
	plaintexts [][]byte,
	metadatas []map[string]any,
	expiresAt *time.Time,
) (result []*tokenizationDomain.Token, err error) {
	start := time.Now()
	result, err = m.inner.TokenizeBatch(ctx, keyName, plaintexts, metadatas, expiresAt)
	metrics.Record(ctx, m.bm, m.domain, "tokenize_batch", start, err)
	return
}

func (m *metricsTokenizationUseCase) Detokenize(
	ctx context.Context,
	token string,
) (plaintext []byte, metadata map[string]any, err error) {
	start := time.Now()
	plaintext, metadata, err = m.inner.Detokenize(ctx, token)
	metrics.Record(ctx, m.bm, m.domain, "detokenize", start, err)
	return
}

func (m *metricsTokenizationUseCase) DetokenizeBatch(
	ctx context.Context,
	tokens []string,
) (plaintexts [][]byte, metadatas []map[string]any, err error) {
	start := time.Now()
	plaintexts, metadatas, err = m.inner.DetokenizeBatch(ctx, tokens)
	metrics.Record(ctx, m.bm, m.domain, "detokenize_batch", start, err)
	return
}

func (m *metricsTokenizationUseCase) Validate(
	ctx context.Context,
	token string,
) (valid bool, err error) {
	start := time.Now()
	valid, err = m.inner.Validate(ctx, token)
	metrics.Record(ctx, m.bm, m.domain, "tokenize_validate", start, err)
	return
}

func (m *metricsTokenizationUseCase) Revoke(ctx context.Context, token string) (err error) {
	start := time.Now()
	err = m.inner.Revoke(ctx, token)
	metrics.Record(ctx, m.bm, m.domain, "tokenize_revoke", start, err)
	return
}

func (m *metricsTokenizationUseCase) CleanupExpired(
	ctx context.Context,
	days int,
	dryRun bool,
) (count int64, err error) {
	start := time.Now()
	count, err = m.inner.CleanupExpired(ctx, days, dryRun)
	metrics.Record(ctx, m.bm, m.domain, "tokenize_cleanup_expired", start, err)
	return
}

var _ TokenizationUseCase = (*metricsTokenizationUseCase)(nil)
