package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

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
func (t *tokenizationKeyUseCaseWithMetrics) Delete(ctx context.Context, tokenizationKeyID uuid.UUID) error {
	start := time.Now()
	err := t.next.Delete(ctx, tokenizationKeyID)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenization_key_delete", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenization_key_delete", time.Since(start), status)

	return err
}
