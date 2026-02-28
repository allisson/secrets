package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func TestRunCreateClient(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	clientID := uuid.New()
	plainSecret := "test-secret"

	t.Run("non-interactive-text", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		input := &authDomain.CreateClientInput{
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{Path: "*", Capabilities: []authDomain.Capability{"read"}},
			},
		}
		output := &authDomain.CreateClientOutput{
			ID:          clientID,
			PlainSecret: plainSecret,
		}

		mockUseCase.On("Create", ctx, input).Return(output, nil)

		var out bytes.Buffer
		io := IOTuple{
			Reader: nil,
			Writer: &out,
		}

		err := RunCreateClient(
			ctx,
			mockUseCase,
			logger,
			"test-client",
			true,
			`[{"path":"*","capabilities":["read"]}]`,
			"text",
			io,
		)

		require.NoError(t, err)
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), plainSecret)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("interactive-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		input := &authDomain.CreateClientInput{
			Name:     "test-client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{Path: "secret/*", Capabilities: []authDomain.Capability{"read", "write"}},
			},
		}
		output := &authDomain.CreateClientOutput{
			ID:          clientID,
			PlainSecret: plainSecret,
		}

		mockUseCase.On("Create", ctx, input).Return(output, nil)

		// Simulate interactive input:
		// 1. Path: secret/*
		// 2. Caps: read,write
		// 3. Add another: n
		userInput := "secret/*\nread,write\nn\n"
		var out bytes.Buffer
		io := IOTuple{
			Reader: bytes.NewBufferString(userInput),
			Writer: &out,
		}

		err := RunCreateClient(ctx, mockUseCase, logger, "test-client", true, "", "json", io)

		require.NoError(t, err)
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), plainSecret)
		require.Contains(t, out.String(), "{") // Should be JSON
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-policies-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockClientUseCase{}
		io := IOTuple{
			Reader: nil,
			Writer: &bytes.Buffer{},
		}

		err := RunCreateClient(ctx, mockUseCase, logger, "test-client", true, `invalid-json`, "text", io)

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse policies JSON")
	})
}
