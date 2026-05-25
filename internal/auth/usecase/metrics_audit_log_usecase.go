package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/metrics"
)

type metricsAuditLogUseCase struct {
	inner  AuditLogUseCase
	bm     metrics.BusinessMetrics
	domain string
}

// NewMetricsAuditLogUseCase wraps inner with per-method timing and operation recording.
func NewMetricsAuditLogUseCase(
	inner AuditLogUseCase,
	bm metrics.BusinessMetrics,
	domain string,
) AuditLogUseCase {
	return &metricsAuditLogUseCase{inner: inner, bm: bm, domain: domain}
}

func (m *metricsAuditLogUseCase) Create(
	ctx context.Context,
	requestID uuid.UUID,
	clientID uuid.UUID,
	capability authDomain.Capability,
	path string,
	metadata map[string]any,
) (err error) {
	start := time.Now()
	err = m.inner.Create(ctx, requestID, clientID, capability, path, metadata)
	metrics.Record(ctx, m.bm, m.domain, "audit_log_create", start, err)
	return
}

func (m *metricsAuditLogUseCase) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
	createdAtFrom, createdAtTo *time.Time,
	clientID *uuid.UUID,
) (result []*authDomain.AuditLog, err error) {
	start := time.Now()
	result, err = m.inner.ListCursor(ctx, afterID, limit, createdAtFrom, createdAtTo, clientID)
	metrics.Record(ctx, m.bm, m.domain, "audit_log_list", start, err)
	return
}

func (m *metricsAuditLogUseCase) DeleteOlderThan(
	ctx context.Context,
	days int,
	dryRun bool,
) (count int64, err error) {
	start := time.Now()
	count, err = m.inner.DeleteOlderThan(ctx, days, dryRun)
	metrics.Record(ctx, m.bm, m.domain, "audit_log_delete", start, err)
	return
}

func (m *metricsAuditLogUseCase) VerifyIntegrity(ctx context.Context, id uuid.UUID) (err error) {
	start := time.Now()
	err = m.inner.VerifyIntegrity(ctx, id)
	metrics.Record(ctx, m.bm, m.domain, "audit_log_verify", start, err)
	return
}

func (m *metricsAuditLogUseCase) VerifyBatch(
	ctx context.Context,
	startTime, endTime time.Time,
) (result *VerificationReport, err error) {
	start := time.Now()
	result, err = m.inner.VerifyBatch(ctx, startTime, endTime)
	metrics.Record(ctx, m.bm, m.domain, "audit_log_verify_batch", start, err)
	return
}

var _ AuditLogUseCase = (*metricsAuditLogUseCase)(nil)
