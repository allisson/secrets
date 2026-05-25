package usecase

import (
	"context"
	"time"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/metrics"
)

type metricsTokenUseCase struct {
	inner  TokenUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsTokenUseCase wraps inner with per-method timing and operation recording.
func NewMetricsTokenUseCase(inner TokenUseCase, bm metrics.BusinessMetrics, domain string) TokenUseCase {
	return &metricsTokenUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsTokenUseCase) Issue(
	ctx context.Context,
	input *authDomain.IssueTokenInput,
) (result *authDomain.IssueTokenOutput, err error) {
	start := time.Now()
	result, err = m.inner.Issue(ctx, input)
	metrics.Record(ctx, m.bm, m.domain, "token_issue", start, err)
	return
}

func (m *metricsTokenUseCase) Authenticate(
	ctx context.Context,
	rawToken string,
) (result *authDomain.Client, err error) {
	start := time.Now()
	result, err = m.inner.Authenticate(ctx, rawToken)
	metrics.Record(ctx, m.bm, m.domain, "token_authenticate", start, err)
	return
}

func (m *metricsTokenUseCase) Revoke(ctx context.Context, rawToken string) (err error) {
	start := time.Now()
	err = m.inner.Revoke(ctx, rawToken)
	metrics.Record(ctx, m.bm, m.domain, "token_revoke", start, err)
	return
}

func (m *metricsTokenUseCase) PurgeExpiredAndRevoked(ctx context.Context, days int) (count int64, err error) {
	start := time.Now()
	count, err = m.inner.PurgeExpiredAndRevoked(ctx, days)
	metrics.Record(ctx, m.bm, m.domain, "token_purge", start, err)
	return
}

var _ TokenUseCase = (*metricsTokenUseCase)(nil)
