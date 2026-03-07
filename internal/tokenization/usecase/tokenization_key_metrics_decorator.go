package usecase

import (
	"context"
	"time"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/metrics"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// tokenizationKeyUseCaseWithMetrics decorates TokenizationKeyUseCase with metrics instrumentation.
type tokenizationKeyUseCaseWithMetrics struct {
	next    TokenizationKeyUseCase
	metrics metrics.BusinessMetrics
}

// NewTokenizationKeyUseCaseWithMetrics wraps a TokenizationKeyUseCase with metrics recording.
func NewTokenizationKeyUseCaseWithMetrics(
	useCase TokenizationKeyUseCase,
	m metrics.BusinessMetrics,
) TokenizationKeyUseCase {
	return &tokenizationKeyUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

// Create records metrics for tokenization key creation operations.
func (t *tokenizationKeyUseCaseWithMetrics) Create(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg cryptoDomain.Algorithm,
) (*tokenizationDomain.TokenizationKey, error) {
	start := time.Now()
	key, err := t.next.Create(ctx, name, formatType, isDeterministic, alg)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_create", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_create", time.Since(start), status)

	return key, err
}

// Rotate records metrics for tokenization key rotation operations.
func (t *tokenizationKeyUseCaseWithMetrics) Rotate(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg cryptoDomain.Algorithm,
) (*tokenizationDomain.TokenizationKey, error) {
	start := time.Now()
	key, err := t.next.Rotate(ctx, name, formatType, isDeterministic, alg)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_rotate", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_rotate", time.Since(start), status)

	return key, err
}

// Delete records metrics for tokenization key deletion operations.
func (t *tokenizationKeyUseCaseWithMetrics) Delete(ctx context.Context, name string) error {
	start := time.Now()
	err := t.next.Delete(ctx, name)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_delete", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_delete", time.Since(start), status)

	return err
}

// GetByName records metrics for tokenization key retrieval operations.
func (t *tokenizationKeyUseCaseWithMetrics) GetByName(
	ctx context.Context,
	name string,
) (*tokenizationDomain.TokenizationKey, error) {
	start := time.Now()
	key, err := t.next.GetByName(ctx, name)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_get", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_get", time.Since(start), status)

	return key, err
}

// List records metrics for tokenization key listing operations.
func (t *tokenizationKeyUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	start := time.Now()
	keys, err := t.next.ListCursor(ctx, afterName, limit)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_list", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_list", time.Since(start), status)

	return keys, err
}

// PurgeDeleted records metrics for tokenization key purge operations.
func (t *tokenizationKeyUseCaseWithMetrics) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (int64, error) {
	start := time.Now()
	count, err := t.next.PurgeDeleted(ctx, olderThanDays, dryRun)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_purge", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_purge", time.Since(start), status)

	return count, err
}
