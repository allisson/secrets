package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/metrics"
)

type metricsClientUseCase struct {
	inner  ClientUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsClientUseCase wraps inner with per-method timing and operation recording.
func NewMetricsClientUseCase(inner ClientUseCase, bm metrics.BusinessMetrics, domain string) ClientUseCase {
	return &metricsClientUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsClientUseCase) Create(
	ctx context.Context,
	input *authDomain.CreateClientInput,
) (result *authDomain.CreateClientOutput, err error) {
	start := time.Now()
	result, err = m.inner.Create(ctx, input)
	metrics.Record(ctx, m.bm, m.domain, "client_create", start, err)
	return
}

func (m *metricsClientUseCase) Update(
	ctx context.Context,
	clientID uuid.UUID,
	input *authDomain.UpdateClientInput,
) (err error) {
	start := time.Now()
	err = m.inner.Update(ctx, clientID, input)
	metrics.Record(ctx, m.bm, m.domain, "client_update", start, err)
	return
}

func (m *metricsClientUseCase) Get(
	ctx context.Context,
	clientID uuid.UUID,
) (result *authDomain.Client, err error) {
	start := time.Now()
	result, err = m.inner.Get(ctx, clientID)
	metrics.Record(ctx, m.bm, m.domain, "client_get", start, err)
	return
}

func (m *metricsClientUseCase) Delete(ctx context.Context, clientID uuid.UUID) (err error) {
	start := time.Now()
	err = m.inner.Delete(ctx, clientID)
	metrics.Record(ctx, m.bm, m.domain, "client_delete", start, err)
	return
}

func (m *metricsClientUseCase) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
) (result []*authDomain.Client, err error) {
	start := time.Now()
	result, err = m.inner.ListCursor(ctx, afterID, limit)
	metrics.Record(ctx, m.bm, m.domain, "client_list", start, err)
	return
}

func (m *metricsClientUseCase) Unlock(ctx context.Context, clientID uuid.UUID) (err error) {
	start := time.Now()
	err = m.inner.Unlock(ctx, clientID)
	metrics.Record(ctx, m.bm, m.domain, "client_unlock", start, err)
	return
}

func (m *metricsClientUseCase) RevokeTokens(ctx context.Context, clientID uuid.UUID) (err error) {
	start := time.Now()
	err = m.inner.RevokeTokens(ctx, clientID)
	metrics.Record(ctx, m.bm, m.domain, "client_revoke_tokens", start, err)
	return
}

func (m *metricsClientUseCase) RotateSecret(
	ctx context.Context,
	clientID uuid.UUID,
) (result *authDomain.CreateClientOutput, err error) {
	start := time.Now()
	result, err = m.inner.RotateSecret(ctx, clientID)
	metrics.Record(ctx, m.bm, m.domain, "client_rotate_secret", start, err)
	return
}

var _ ClientUseCase = (*metricsClientUseCase)(nil)
