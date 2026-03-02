package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func TestRunCreateClient(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	name := "test-client"
	policies := []authDomain.PolicyDocument{
		{
			Path:         "*",
			Capabilities: []authDomain.Capability{"read"},
		},
	}
	policiesJSON, _ := json.Marshal(policies)
	clientID := uuid.New()
	plainSecret := "plain-secret"

	t.Run("non-interactive-text", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Create", ctx, mock.Anything).Return(&authDomain.CreateClientOutput{
			ID:          clientID,
			PlainSecret: plainSecret,
		}, nil)

		var out bytes.Buffer
		io := IOTuple{Writer: &out}
		err := RunCreateClient(ctx, mockUseCase, logger, name, true, string(policiesJSON), "text", io)

		require.NoError(t, err)
		require.Contains(t, out.String(), "Client created successfully!")
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), plainSecret)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("non-interactive-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Create", ctx, mock.Anything).Return(&authDomain.CreateClientOutput{
			ID:          clientID,
			PlainSecret: plainSecret,
		}, nil)

		var out bytes.Buffer
		io := IOTuple{Writer: &out}
		err := RunCreateClient(ctx, mockUseCase, logger, name, true, string(policiesJSON), "json", io)

		require.NoError(t, err)
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), plainSecret)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-policies-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		err := RunCreateClient(ctx, mockUseCase, logger, name, true, "invalid-json", "text", IOTuple{})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse policies JSON")
	})

	t.Run("empty-policies", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		err := RunCreateClient(ctx, mockUseCase, logger, name, true, "[]", "text", IOTuple{})

		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one policy is required")
	})
}
