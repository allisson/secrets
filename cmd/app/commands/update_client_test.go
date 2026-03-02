package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func TestRunUpdateClient(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	clientID := uuid.New()
	name := "updated-client"
	policies := []authDomain.PolicyDocument{
		{
			Path:         "*",
			Capabilities: []authDomain.Capability{"read", "write"},
		},
	}
	policiesJSON, _ := json.Marshal(policies)

	t.Run("non-interactive-text", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(&authDomain.Client{ID: clientID}, nil)
		mockUseCase.On("Update", ctx, clientID, mock.Anything).Return(nil)

		var out bytes.Buffer
		io := IOTuple{Writer: &out}
		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			io,
			clientID.String(),
			name,
			true,
			string(policiesJSON),
			"text",
		)

		require.NoError(t, err)
		require.Contains(t, out.String(), "Client updated successfully!")
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), name)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("non-interactive-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(&authDomain.Client{ID: clientID}, nil)
		mockUseCase.On("Update", ctx, clientID, mock.Anything).Return(nil)

		var out bytes.Buffer
		io := IOTuple{Writer: &out}
		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			io,
			clientID.String(),
			name,
			true,
			string(policiesJSON),
			"json",
		)

		require.NoError(t, err)
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), name)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-id", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			IOTuple{},
			"invalid-id",
			name,
			true,
			string(policiesJSON),
			"text",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid client ID format")
	})

	t.Run("not-found", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(nil, fmt.Errorf("not found"))

		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			IOTuple{},
			clientID.String(),
			name,
			true,
			string(policiesJSON),
			"text",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get existing client")
	})
}
