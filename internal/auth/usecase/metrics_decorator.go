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

// Create records metrics for client creation operations.
func (c *clientUseCaseWithMetrics) Create(
	ctx context.Context,
	createClientInput *authDomain.CreateClientInput,
) (*authDomain.CreateClientOutput, error) {
	start := time.Now()
	output, err := c.next.Create(ctx, createClientInput)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_create", status)
	c.metrics.RecordDuration(ctx, "auth", "client_create", time.Since(start), status)

	return output, err
}

// Update records metrics for client update operations.
func (c *clientUseCaseWithMetrics) Update(
	ctx context.Context,
	clientID uuid.UUID,
	updateClientInput *authDomain.UpdateClientInput,
) error {
	start := time.Now()
	err := c.next.Update(ctx, clientID, updateClientInput)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_update", status)
	c.metrics.RecordDuration(ctx, "auth", "client_update", time.Since(start), status)

	return err
}

// Get records metrics for client retrieval operations.
func (c *clientUseCaseWithMetrics) Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error) {
	start := time.Now()
	client, err := c.next.Get(ctx, clientID)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_get", status)
	c.metrics.RecordDuration(ctx, "auth", "client_get", time.Since(start), status)

	return client, err
}

// List records metrics for client list operations.
func (c *clientUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
) ([]*authDomain.Client, error) {
	start := time.Now()
	clients, err := c.next.ListCursor(ctx, afterID, limit)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_list", status)
	c.metrics.RecordDuration(ctx, "auth", "client_list", time.Since(start), status)

	return clients, err
}

// Delete records metrics for client deletion operations.
func (c *clientUseCaseWithMetrics) Delete(ctx context.Context, clientID uuid.UUID) error {
	start := time.Now()
	err := c.next.Delete(ctx, clientID)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_delete", status)
	c.metrics.RecordDuration(ctx, "auth", "client_delete", time.Since(start), status)

	return err
}

// Unlock records metrics for client unlock operations.
func (c *clientUseCaseWithMetrics) Unlock(ctx context.Context, clientID uuid.UUID) error {
	start := time.Now()
	err := c.next.Unlock(ctx, clientID)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_unlock", status)
	c.metrics.RecordDuration(ctx, "auth", "client_unlock", time.Since(start), status)

	return err
}

// RevokeTokens records metrics for client token revocation operations.
func (c *clientUseCaseWithMetrics) RevokeTokens(ctx context.Context, clientID uuid.UUID) error {
	start := time.Now()
	err := c.next.RevokeTokens(ctx, clientID)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_revoke_tokens", status)
	c.metrics.RecordDuration(ctx, "auth", "client_revoke_tokens", time.Since(start), status)

	return err
}

// RotateSecret records metrics for client secret rotation operations.
func (c *clientUseCaseWithMetrics) RotateSecret(
	ctx context.Context,
	clientID uuid.UUID,
) (*authDomain.CreateClientOutput, error) {
	start := time.Now()
	output, err := c.next.RotateSecret(ctx, clientID)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordOperation(ctx, "auth", "client_rotate_secret", status)
	c.metrics.RecordDuration(ctx, "auth", "client_rotate_secret", time.Since(start), status)

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

// Issue records metrics for token issuance operations.
func (t *tokenUseCaseWithMetrics) Issue(
	ctx context.Context,
	issueTokenInput *authDomain.IssueTokenInput,
) (*authDomain.IssueTokenOutput, error) {
	start := time.Now()
	output, err := t.next.Issue(ctx, issueTokenInput)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "auth", "token_issue", status)
	t.metrics.RecordDuration(ctx, "auth", "token_issue", time.Since(start), status)

	return output, err
}

// Authenticate records metrics for token authentication operations.
func (t *tokenUseCaseWithMetrics) Authenticate(
	ctx context.Context,
	tokenHash string,
) (*authDomain.Client, error) {
	start := time.Now()
	client, err := t.next.Authenticate(ctx, tokenHash)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "auth", "token_authenticate", status)
	t.metrics.RecordDuration(ctx, "auth", "token_authenticate", time.Since(start), status)

	return client, err
}

// Revoke records metrics for token revocation operations.
func (t *tokenUseCaseWithMetrics) Revoke(ctx context.Context, tokenHash string) error {
	start := time.Now()
	err := t.next.Revoke(ctx, tokenHash)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "auth", "token_revoke", status)
	t.metrics.RecordDuration(ctx, "auth", "token_revoke", time.Since(start), status)

	return err
}

// PurgeExpiredAndRevoked records metrics for token purging operations.
func (t *tokenUseCaseWithMetrics) PurgeExpiredAndRevoked(ctx context.Context, days int) (int64, error) {
	start := time.Now()
	count, err := t.next.PurgeExpiredAndRevoked(ctx, days)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "auth", "token_purge", status)
	t.metrics.RecordDuration(ctx, "auth", "token_purge", time.Since(start), status)

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

// Create records metrics for audit log creation operations.
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

	status := "success"
	if err != nil {
		status = "error"
	}

	a.metrics.RecordOperation(ctx, "auth", "audit_log_create", status)
	a.metrics.RecordDuration(ctx, "auth", "audit_log_create", time.Since(start), status)

	return err
}

// List records metrics for audit log list operations.
func (a *auditLogUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
	createdAtFrom, createdAtTo *time.Time,
) ([]*authDomain.AuditLog, error) {
	start := time.Now()
	logs, err := a.next.ListCursor(ctx, afterID, limit, createdAtFrom, createdAtTo)

	status := "success"
	if err != nil {
		status = "error"
	}

	a.metrics.RecordOperation(ctx, "auth", "audit_log_list", status)
	a.metrics.RecordDuration(ctx, "auth", "audit_log_list", time.Since(start), status)

	return logs, err
}

// DeleteOlderThan records metrics for audit log deletion operations.
func (a *auditLogUseCaseWithMetrics) DeleteOlderThan(
	ctx context.Context,
	days int,
	dryRun bool,
) (int64, error) {
	start := time.Now()
	count, err := a.next.DeleteOlderThan(ctx, days, dryRun)

	status := "success"
	if err != nil {
		status = "error"
	}

	a.metrics.RecordOperation(ctx, "auth", "audit_log_delete", status)
	a.metrics.RecordDuration(ctx, "auth", "audit_log_delete", time.Since(start), status)

	return count, err
}

// VerifyIntegrity records metrics for single audit log verification operations.
func (a *auditLogUseCaseWithMetrics) VerifyIntegrity(
	ctx context.Context,
	id uuid.UUID,
) error {
	start := time.Now()
	err := a.next.VerifyIntegrity(ctx, id)

	status := "success"
	if err != nil {
		status = "error"
	}

	a.metrics.RecordOperation(ctx, "auth", "audit_log_verify", status)
	a.metrics.RecordDuration(ctx, "auth", "audit_log_verify", time.Since(start), status)

	return err
}

// VerifyBatch records metrics for batch audit log verification operations.
func (a *auditLogUseCaseWithMetrics) VerifyBatch(
	ctx context.Context,
	startTime, endTime time.Time,
) (*VerificationReport, error) {
	start := time.Now()
	report, err := a.next.VerifyBatch(ctx, startTime, endTime)

	status := "success"
	if err != nil {
		status = "error"
	}

	a.metrics.RecordOperation(ctx, "auth", "audit_log_verify_batch", status)
	a.metrics.RecordDuration(ctx, "auth", "audit_log_verify_batch", time.Since(start), status)

	return report, err
}
