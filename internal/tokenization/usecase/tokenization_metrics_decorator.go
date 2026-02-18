package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/metrics"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// tokenizationUseCaseWithMetrics decorates TokenizationUseCase with metrics instrumentation.
type tokenizationUseCaseWithMetrics struct {
	next    TokenizationUseCase
	metrics metrics.BusinessMetrics
}

// NewTokenizationUseCaseWithMetrics wraps a TokenizationUseCase with metrics recording.
func NewTokenizationUseCaseWithMetrics(
	useCase TokenizationUseCase,
	m metrics.BusinessMetrics,
) TokenizationUseCase {
	return &tokenizationUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

// Tokenize records metrics for token generation operations.
func (t *tokenizationUseCaseWithMetrics) Tokenize(
	ctx context.Context,
	keyName string,
	plaintext []byte,
	metadata map[string]any,
	expiresAt *time.Time,
) (*tokenizationDomain.Token, error) {
	start := time.Now()
	token, err := t.next.Tokenize(ctx, keyName, plaintext, metadata, expiresAt)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "tokenize", status)
	t.metrics.RecordDuration(ctx, "tokenization", "tokenize", time.Since(start), status)

	return token, err
}

// Detokenize records metrics for token detokenization operations.
func (t *tokenizationUseCaseWithMetrics) Detokenize(
	ctx context.Context,
	token string,
) (plaintext []byte, metadata map[string]any, err error) {
	start := time.Now()
	plaintext, metadata, err = t.next.Detokenize(ctx, token)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "detokenize", status)
	t.metrics.RecordDuration(ctx, "tokenization", "detokenize", time.Since(start), status)

	return plaintext, metadata, err
}

// Validate records metrics for token validation operations.
func (t *tokenizationUseCaseWithMetrics) Validate(ctx context.Context, token string) (bool, error) {
	start := time.Now()
	valid, err := t.next.Validate(ctx, token)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "validate", status)
	t.metrics.RecordDuration(ctx, "tokenization", "validate", time.Since(start), status)

	return valid, err
}

// Revoke records metrics for token revocation operations.
func (t *tokenizationUseCaseWithMetrics) Revoke(ctx context.Context, token string) error {
	start := time.Now()
	err := t.next.Revoke(ctx, token)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "revoke", status)
	t.metrics.RecordDuration(ctx, "tokenization", "revoke", time.Since(start), status)

	return err
}

// CleanupExpired records metrics for expired token cleanup operations.
func (t *tokenizationUseCaseWithMetrics) CleanupExpired(
	ctx context.Context,
	days int,
	dryRun bool,
) (int64, error) {
	start := time.Now()
	count, err := t.next.CleanupExpired(ctx, days, dryRun)

	status := "success"
	if err != nil {
		status = "error"
	}

	t.metrics.RecordOperation(ctx, "tokenization", "cleanup_expired", status)
	t.metrics.RecordDuration(ctx, "tokenization", "cleanup_expired", time.Since(start), status)

	return count, err
}
