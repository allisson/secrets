package usecase

import (
	"context"
	"time"

	"github.com/allisson/secrets/internal/metrics"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// secretUseCaseWithMetrics decorates SecretUseCase with metrics instrumentation.
type secretUseCaseWithMetrics struct {
	next    SecretUseCase
	metrics metrics.BusinessMetrics
}

// NewSecretUseCaseWithMetrics wraps a SecretUseCase with metrics recording.
func NewSecretUseCaseWithMetrics(useCase SecretUseCase, m metrics.BusinessMetrics) SecretUseCase {
	return &secretUseCaseWithMetrics{
		next:    useCase,
		metrics: m,
	}
}

// CreateOrUpdate records metrics for secret creation/update operations.
func (s *secretUseCaseWithMetrics) CreateOrUpdate(
	ctx context.Context,
	path string,
	value []byte,
) (*secretsDomain.Secret, error) {
	start := time.Now()
	secret, err := s.next.CreateOrUpdate(ctx, path, value)

	status := "success"
	if err != nil {
		status = "error"
	}

	s.metrics.RecordOperation(ctx, "secrets", "secret_create", status)
	s.metrics.RecordDuration(ctx, "secrets", "secret_create", time.Since(start), status)

	return secret, err
}

// Get records metrics for secret retrieval operations.
func (s *secretUseCaseWithMetrics) Get(ctx context.Context, path string) (*secretsDomain.Secret, error) {
	start := time.Now()
	secret, err := s.next.Get(ctx, path)

	status := "success"
	if err != nil {
		status = "error"
	}

	s.metrics.RecordOperation(ctx, "secrets", "secret_get", status)
	s.metrics.RecordDuration(ctx, "secrets", "secret_get", time.Since(start), status)

	return secret, err
}

// GetByVersion records metrics for versioned secret retrieval operations.
func (s *secretUseCaseWithMetrics) GetByVersion(
	ctx context.Context,
	path string,
	version uint,
) (*secretsDomain.Secret, error) {
	start := time.Now()
	secret, err := s.next.GetByVersion(ctx, path, version)

	status := "success"
	if err != nil {
		status = "error"
	}

	s.metrics.RecordOperation(ctx, "secrets", "secret_get_version", status)
	s.metrics.RecordDuration(ctx, "secrets", "secret_get_version", time.Since(start), status)

	return secret, err
}

// Delete records metrics for secret deletion operations.
func (s *secretUseCaseWithMetrics) Delete(ctx context.Context, path string) error {
	start := time.Now()
	err := s.next.Delete(ctx, path)

	status := "success"
	if err != nil {
		status = "error"
	}

	s.metrics.RecordOperation(ctx, "secrets", "secret_delete", status)
	s.metrics.RecordDuration(ctx, "secrets", "secret_delete", time.Since(start), status)

	return err
}
