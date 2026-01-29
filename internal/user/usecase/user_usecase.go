// Package usecase implements the user business logic and orchestrates user domain operations.
package usecase

import (
	"context"
	"encoding/json"
	"strings"

	validation "github.com/jellydator/validation"

	"github.com/allisson/go-pwdhash"
	"github.com/google/uuid"

	"github.com/allisson/go-project-template/internal/database"
	apperrors "github.com/allisson/go-project-template/internal/errors"
	outboxDomain "github.com/allisson/go-project-template/internal/outbox/domain"
	"github.com/allisson/go-project-template/internal/user/domain"
	appValidation "github.com/allisson/go-project-template/internal/validation"
)

// RegisterUserInput contains the input data for user registration
type RegisterUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UseCase defines the interface for user business logic operations
type UseCase interface {
	RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// UserRepository interface defines user repository operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// OutboxEventRepository interface defines outbox event repository operations
type OutboxEventRepository interface {
	Create(ctx context.Context, event *outboxDomain.OutboxEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*outboxDomain.OutboxEvent, error)
	Update(ctx context.Context, event *outboxDomain.OutboxEvent) error
}

// UserUseCase handles user-related business logic
type UserUseCase struct {
	txManager      database.TxManager
	userRepo       UserRepository
	outboxRepo     OutboxEventRepository
	passwordHasher *pwdhash.PasswordHasher
}

// NewUserUseCase creates a new UserUseCase
func NewUserUseCase(
	txManager database.TxManager,
	userRepo UserRepository,
	outboxRepo OutboxEventRepository,
) (UseCase, error) {
	// Initialize password hasher with interactive policy for user passwords
	hasher, err := pwdhash.New(pwdhash.WithPolicy(pwdhash.PolicyInteractive))
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to create password hasher")
	}

	return &UserUseCase{
		txManager:      txManager,
		userRepo:       userRepo,
		outboxRepo:     outboxRepo,
		passwordHasher: hasher,
	}, nil
}

// validateRegisterUserInput validates the registration input using jellydator/validation
// This provides comprehensive validation including:
// - Required field checks
// - Email format validation
// - Password strength requirements (min 8 chars, uppercase, lowercase, number, special char)
func (uc *UserUseCase) validateRegisterUserInput(input RegisterUserInput) error {
	err := validation.ValidateStruct(&input,
		validation.Field(&input.Name,
			validation.Required.Error("name is required"),
			appValidation.NotBlank,
			validation.Length(1, 255).Error("name must be between 1 and 255 characters"),
		),
		validation.Field(&input.Email,
			validation.Required.Error("email is required"),
			appValidation.NotBlank,
			appValidation.Email,
			validation.Length(5, 255).Error("email must be between 5 and 255 characters"),
		),
		validation.Field(&input.Password,
			validation.Required.Error("password is required"),
			validation.Length(8, 128).Error("password must be between 8 and 128 characters"),
			appValidation.PasswordStrength{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			},
		),
	)
	return appValidation.WrapValidationError(err)
}

// RegisterUser registers a new user and creates a user.created event
func (uc *UserUseCase) RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
	// Validate input
	if err := uc.validateRegisterUserInput(input); err != nil {
		return nil, err
	}

	// Hash the password
	hashedPassword, err := uc.passwordHasher.Hash([]byte(input.Password))
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to hash password")
	}

	user := &domain.User{
		ID:       uuid.Must(uuid.NewV7()),
		Name:     strings.TrimSpace(input.Name),
		Email:    strings.TrimSpace(strings.ToLower(input.Email)),
		Password: hashedPassword,
	}

	// Execute within a transaction
	err = uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		// Create user - repository will return domain errors
		if err := uc.userRepo.Create(ctx, user); err != nil {
			return err
		}

		// Create user.created event payload
		eventPayload := map[string]interface{}{
			"user_id": user.ID,
			"name":    user.Name,
			"email":   user.Email,
		}
		payloadJSON, err := json.Marshal(eventPayload)
		if err != nil {
			return apperrors.Wrap(err, "failed to marshal event payload")
		}

		// Create outbox event
		outboxEvent := &outboxDomain.OutboxEvent{
			ID:        uuid.Must(uuid.NewV7()),
			EventType: "user.created",
			Payload:   string(payloadJSON),
			Status:    outboxDomain.OutboxEventStatusPending,
			Retries:   0,
		}

		if err := uc.outboxRepo.Create(ctx, outboxEvent); err != nil {
			return apperrors.Wrap(err, "failed to create outbox event")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (uc *UserUseCase) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return uc.userRepo.GetByEmail(ctx, email)
}

// GetUserByID retrieves a user by ID
func (uc *UserUseCase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, id)
}
