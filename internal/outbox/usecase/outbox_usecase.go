// Package usecase implements the outbox business logic and orchestrates outbox domain operations.
package usecase

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/allisson/go-project-template/internal/database"
	"github.com/allisson/go-project-template/internal/outbox/domain"
)

// Config holds outbox use case configuration
type Config struct {
	Interval      time.Duration
	BatchSize     int
	MaxRetries    int
	RetryInterval time.Duration
}

// OutboxEventRepository defines outbox event repository operations
type OutboxEventRepository interface {
	Create(ctx context.Context, event *domain.OutboxEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	Update(ctx context.Context, event *domain.OutboxEvent) error
}

// EventProcessor defines the interface for processing different event types
type EventProcessor interface {
	Process(ctx context.Context, event *domain.OutboxEvent) error
}

// UseCase defines the interface for outbox use cases
type UseCase interface {
	Start(ctx context.Context) error
	ProcessEvents(ctx context.Context) error
}

// OutboxUseCase implements business logic for processing outbox events
type OutboxUseCase struct {
	config         Config
	txManager      database.TxManager
	outboxRepo     OutboxEventRepository
	eventProcessor EventProcessor
	logger         *slog.Logger
}

// NewOutboxUseCase creates a new OutboxUseCase
func NewOutboxUseCase(
	config Config,
	txManager database.TxManager,
	outboxRepo OutboxEventRepository,
	eventProcessor EventProcessor,
	logger *slog.Logger,
) *OutboxUseCase {
	return &OutboxUseCase{
		config:         config,
		txManager:      txManager,
		outboxRepo:     outboxRepo,
		eventProcessor: eventProcessor,
		logger:         logger,
	}
}

// Start starts the outbox event processing loop
func (uc *OutboxUseCase) Start(ctx context.Context) error {
	if uc.logger != nil {
		uc.logger.Info("starting outbox event processor",
			slog.Duration("interval", uc.config.Interval),
			slog.Int("batch_size", uc.config.BatchSize),
		)
	}

	ticker := time.NewTicker(uc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if uc.logger != nil {
				uc.logger.Info("stopping outbox event processor")
			}
			return ctx.Err()
		case <-ticker.C:
			if err := uc.ProcessEvents(ctx); err != nil {
				if uc.logger != nil {
					uc.logger.Error("failed to process events", slog.Any("error", err))
				}
			}
		}
	}
}

// ProcessEvents retrieves and processes pending events from the outbox in a transaction
func (uc *OutboxUseCase) ProcessEvents(ctx context.Context) error {
	return uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		// Get pending events
		events, err := uc.outboxRepo.GetPendingEvents(ctx, uc.config.BatchSize)
		if err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		if uc.logger != nil {
			uc.logger.Info("processing events", slog.Int("count", len(events)))
		}

		for _, event := range events {
			if err := uc.processEvent(ctx, event); err != nil {
				if uc.logger != nil {
					uc.logger.Error("failed to process event",
						slog.String("event_id", event.ID.String()),
						slog.String("event_type", event.EventType),
						slog.Any("error", err),
					)
				}

				// Update event as failed
				event.Retries++
				errorMsg := err.Error()
				event.LastError = &errorMsg

				if event.Retries >= uc.config.MaxRetries {
					event.Status = domain.OutboxEventStatusFailed
				}

				if err := uc.outboxRepo.Update(ctx, event); err != nil {
					return err
				}
				continue
			}

			// Mark event as processed
			now := time.Now()
			event.Status = domain.OutboxEventStatusProcessed
			event.ProcessedAt = &now

			if err := uc.outboxRepo.Update(ctx, event); err != nil {
				return err
			}
		}

		return nil
	})
}

// processEvent handles a single outbox event using the configured event processor
func (uc *OutboxUseCase) processEvent(ctx context.Context, event *domain.OutboxEvent) error {
	if uc.logger != nil {
		uc.logger.Info("processing event",
			slog.String("event_id", event.ID.String()),
			slog.String("event_type", event.EventType),
		)
	}

	return uc.eventProcessor.Process(ctx, event)
}

// DefaultEventProcessor is a default implementation of EventProcessor
type DefaultEventProcessor struct {
	logger *slog.Logger
}

// NewDefaultEventProcessor creates a new DefaultEventProcessor
func NewDefaultEventProcessor(logger *slog.Logger) *DefaultEventProcessor {
	return &DefaultEventProcessor{
		logger: logger,
	}
}

// Process handles event processing with basic logging
func (p *DefaultEventProcessor) Process(ctx context.Context, event *domain.OutboxEvent) error {
	// Parse event payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		return err
	}

	// Handle different event types
	switch event.EventType {
	case "user.created":
		if p.logger != nil {
			p.logger.Info("user created event",
				slog.Any("payload", payload),
			)
		}
		// In a real application, you might publish this to a message queue,
		// send notifications, update cache, etc.
	default:
		if p.logger != nil {
			p.logger.Warn("unknown event type", slog.String("event_type", event.EventType))
		}
	}

	return nil
}
