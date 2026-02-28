package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	clientIDStr := clientID.String()

	existingClient := &authDomain.Client{
		ID:       clientID,
		Name:     "old-name",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{Path: "secret/*", Capabilities: []authDomain.Capability{authDomain.ReadCapability}},
		},
	}

	t.Run("success-non-interactive-text", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(existingClient, nil)
		mockUseCase.On("Update", ctx, clientID, mock.AnythingOfType("*domain.UpdateClientInput")).Return(nil)

		var out bytes.Buffer
		io := IOTuple{Reader: &bytes.Buffer{}, Writer: &out}

		policiesJSON := `[{"path": "secret/*", "capabilities": ["read", "write"]}]`
		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			io,
			clientIDStr,
			"new-name",
			true,
			policiesJSON,
			"text",
		)

		require.NoError(t, err)
		require.Contains(t, out.String(), "Client updated successfully!")
		require.Contains(t, out.String(), "Name: new-name")

		mockUseCase.AssertExpectations(t)
	})

	t.Run("success-non-interactive-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(existingClient, nil)
		mockUseCase.On("Update", ctx, clientID, mock.AnythingOfType("*domain.UpdateClientInput")).Return(nil)

		var out bytes.Buffer
		io := IOTuple{Reader: &bytes.Buffer{}, Writer: &out}

		policiesJSON := `[{"path": "secret/*", "capabilities": ["read", "write"]}]`
		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			io,
			clientIDStr,
			"new-name",
			true,
			policiesJSON,
			"json",
		)

		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)
		require.Equal(t, clientIDStr, result["client_id"])
		require.Equal(t, "new-name", result["name"])
		require.Equal(t, true, result["is_active"])

		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-client-id", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		err := RunUpdateClient(
			ctx,
			mockUseCase,
			logger,
			DefaultIO(),
			"invalid-uuid",
			"name",
			true,
			"",
			"text",
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid client ID format")
	})

	t.Run("client-not-found", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(nil, errors.New("client not found"))

		err := RunUpdateClient(ctx, mockUseCase, logger, DefaultIO(), clientIDStr, "name", true, "", "text")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get existing client")
	})

	t.Run("interactive-success", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		mockUseCase.On("Get", ctx, clientID).Return(existingClient, nil)
		mockUseCase.On("Update", ctx, clientID, mock.AnythingOfType("*domain.UpdateClientInput")).Return(nil)

		// Mock user input: path, capabilities, another policy? (n)
		input := bytes.NewBufferString("secret/test\nread,write\nn\n")
		var out bytes.Buffer
		io := IOTuple{Reader: input, Writer: &out}

		err := RunUpdateClient(ctx, mockUseCase, logger, io, clientIDStr, "new-name", true, "", "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Client updated successfully!")

		mockUseCase.AssertExpectations(t)
	})
}
