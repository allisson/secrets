package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/metrics"
)

// clientUseCaseWithMetrics decorates ClientUseCase with metrics instrumentation.
type clientUseCaseWithMetrics struct {
	next    ClientUseCase
	metrics metrics.BusinessMetrics
}

// NewClientUseCaseWithMetrics wraps a ClientUseCase with metrics recording.
func NewClientUseCaseWithMetrics(useCase ClientUseCase, m metrics.BusinessMetrics) ClientUseCase {
	return &clientUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

func (c *clientUseCaseWithMetrics) Create(
	ctx context.Context,
	createClientInput *authDomain.CreateClientInput,
) (*authDomain.CreateClientOutput, error) {
	start := time.Now()
	output, err := c.next.Create(ctx, createClientInput)
	metrics.Record(ctx, c.metrics, "auth", "client_create", start, err)
	return output, err
}

func (c *clientUseCaseWithMetrics) Update(
	ctx context.Context,
	clientID uuid.UUID,
	updateClientInput *authDomain.UpdateClientInput,
) error {
	start := time.Now()
	err := c.next.Update(ctx, clientID, updateClientInput)
	metrics.Record(ctx, c.metrics, "auth", "client_update", start, err)
	return err
}

func (c *clientUseCaseWithMetrics) Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error) {
	start := time.Now()
	client, err := c.next.Get(ctx, clientID)
	metrics.Record(ctx, c.metrics, "auth", "client_get", start, err)
	return client, err
}

func (c *clientUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
) ([]*authDomain.Client, error) {
	start := time.Now()
	clients, err := c.next.ListCursor(ctx, afterID, limit)
	metrics.Record(ctx, c.metrics, "auth", "client_list", start, err)
	return clients, err
}

func (c *clientUseCaseWithMetrics) Delete(ctx context.Context, clientID uuid.UUID) error {
	start := time.Now()
	err := c.next.Delete(ctx, clientID)
	metrics.Record(ctx, c.metrics, "auth", "client_delete", start, err)
	return err
}

func (c *clientUseCaseWithMetrics) Unlock(ctx context.Context, clientID uuid.UUID) error {
	start := time.Now()
	err := c.next.Unlock(ctx, clientID)
	metrics.Record(ctx, c.metrics, "auth", "client_unlock", start, err)
	return err
}

func (c *clientUseCaseWithMetrics) RevokeTokens(ctx context.Context, clientID uuid.UUID) error {
	start := time.Now()
	err := c.next.RevokeTokens(ctx, clientID)
	metrics.Record(ctx, c.metrics, "auth", "client_revoke_tokens", start, err)
	return err
}

func (c *clientUseCaseWithMetrics) RotateSecret(
	ctx context.Context,
	clientID uuid.UUID,
) (*authDomain.CreateClientOutput, error) {
	start := time.Now()
	output, err := c.next.RotateSecret(ctx, clientID)
	metrics.Record(ctx, c.metrics, "auth", "client_rotate_secret", start, err)
	return output, err
}

// tokenUseCaseWithMetrics decorates TokenUseCase with metrics instrumentation.
type tokenUseCaseWithMetrics struct {
	next    TokenUseCase
	metrics metrics.BusinessMetrics
}

// NewTokenUseCaseWithMetrics wraps a TokenUseCase with metrics recording.
func NewTokenUseCaseWithMetrics(useCase TokenUseCase, m metrics.BusinessMetrics) TokenUseCase {
	return &tokenUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

func (t *tokenUseCaseWithMetrics) Issue(
	ctx context.Context,
	issueTokenInput *authDomain.IssueTokenInput,
) (*authDomain.IssueTokenOutput, error) {
	start := time.Now()
	output, err := t.next.Issue(ctx, issueTokenInput)
	metrics.Record(ctx, t.metrics, "auth", "token_issue", start, err)
	return output, err
}

func (t *tokenUseCaseWithMetrics) Authenticate(
	ctx context.Context,
	tokenHash string,
) (*authDomain.Client, error) {
	start := time.Now()
	client, err := t.next.Authenticate(ctx, tokenHash)
	metrics.Record(ctx, t.metrics, "auth", "token_authenticate", start, err)
	return client, err
}

func (t *tokenUseCaseWithMetrics) Revoke(ctx context.Context, tokenHash string) error {
	start := time.Now()
	err := t.next.Revoke(ctx, tokenHash)
	metrics.Record(ctx, t.metrics, "auth", "token_revoke", start, err)
	return err
}

func (t *tokenUseCaseWithMetrics) PurgeExpiredAndRevoked(ctx context.Context, days int) (int64, error) {
	start := time.Now()
	count, err := t.next.PurgeExpiredAndRevoked(ctx, days)
	metrics.Record(ctx, t.metrics, "auth", "token_purge", start, err)
	return count, err
}

// auditLogUseCaseWithMetrics decorates AuditLogUseCase with metrics instrumentation.
type auditLogUseCaseWithMetrics struct {
	next    AuditLogUseCase
	metrics metrics.BusinessMetrics
}

// NewAuditLogUseCaseWithMetrics wraps an AuditLogUseCase with metrics recording.
func NewAuditLogUseCaseWithMetrics(useCase AuditLogUseCase, m metrics.BusinessMetrics) AuditLogUseCase {
	return &auditLogUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

func (a *auditLogUseCaseWithMetrics) Create(
	ctx context.Context,
	requestID uuid.UUID,
	clientID uuid.UUID,
	capability authDomain.Capability,
	path string,
	metadata map[string]any,
) error {
	start := time.Now()
	err := a.next.Create(ctx, requestID, clientID, capability, path, metadata)
	metrics.Record(ctx, a.metrics, "auth", "audit_log_create", start, err)
	return err
}

func (a *auditLogUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
	createdAtFrom, createdAtTo *time.Time,
	clientID *uuid.UUID,
) ([]*authDomain.AuditLog, error) {
	start := time.Now()
	logs, err := a.next.ListCursor(ctx, afterID, limit, createdAtFrom, createdAtTo, clientID)
	metrics.Record(ctx, a.metrics, "auth", "audit_log_list", start, err)
	return logs, err
}

func (a *auditLogUseCaseWithMetrics) DeleteOlderThan(
	ctx context.Context,
	days int,
	dryRun bool,
) (int64, error) {
	start := time.Now()
	count, err := a.next.DeleteOlderThan(ctx, days, dryRun)
	metrics.Record(ctx, a.metrics, "auth", "audit_log_delete", start, err)
	return count, err
}

func (a *auditLogUseCaseWithMetrics) VerifyIntegrity(
	ctx context.Context,
	id uuid.UUID,
) error {
	start := time.Now()
	err := a.next.VerifyIntegrity(ctx, id)
	metrics.Record(ctx, a.metrics, "auth", "audit_log_verify", start, err)
	return err
}

func (a *auditLogUseCaseWithMetrics) VerifyBatch(
	ctx context.Context,
	startTime, endTime time.Time,
) (*VerificationReport, error) {
	start := time.Now()
	report, err := a.next.VerifyBatch(ctx, startTime, endTime)
	metrics.Record(ctx, a.metrics, "auth", "audit_log_verify_batch", start, err)
	return report, err
}
