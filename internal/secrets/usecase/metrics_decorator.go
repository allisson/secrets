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

func (s *secretUseCaseWithMetrics) CreateOrUpdate(
	ctx context.Context,
	path string,
	value []byte,
) (*secretsDomain.Secret, error) {
	start := time.Now()
	secret, err := s.next.CreateOrUpdate(ctx, path, value)
	metrics.Record(ctx, s.metrics, "secrets", "secret_create", start, err)
	return secret, err
}

func (s *secretUseCaseWithMetrics) Get(ctx context.Context, path string) (*secretsDomain.Secret, error) {
	start := time.Now()
	secret, err := s.next.Get(ctx, path)
	metrics.Record(ctx, s.metrics, "secrets", "secret_get", start, err)
	return secret, err
}

func (s *secretUseCaseWithMetrics) GetByVersion(
	ctx context.Context,
	path string,
	version uint,
) (*secretsDomain.Secret, error) {
	start := time.Now()
	secret, err := s.next.GetByVersion(ctx, path, version)
	metrics.Record(ctx, s.metrics, "secrets", "secret_get_version", start, err)
	return secret, err
}

func (s *secretUseCaseWithMetrics) Delete(ctx context.Context, path string) error {
	start := time.Now()
	err := s.next.Delete(ctx, path)
	metrics.Record(ctx, s.metrics, "secrets", "secret_delete", start, err)
	return err
}

func (s *secretUseCaseWithMetrics) ListCursor(
	ctx context.Context,
	afterPath *string,
	limit int,
) ([]*secretsDomain.Secret, error) {
	start := time.Now()
	secrets, err := s.next.ListCursor(ctx, afterPath, limit)
	metrics.Record(ctx, s.metrics, "secrets", "secret_list", start, err)
	return secrets, err
}

func (s *secretUseCaseWithMetrics) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (int64, error) {
	start := time.Now()
	count, err := s.next.PurgeDeleted(ctx, olderThanDays, dryRun)
	metrics.Record(ctx, s.metrics, "secrets", "secret_purge", start, err)
	return count, err
}
