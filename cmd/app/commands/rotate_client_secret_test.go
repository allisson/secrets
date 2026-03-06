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

func TestRunRotateClientSecret(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	clientID := uuid.Must(uuid.NewV7())
	plainSecret := "new-plain-secret"

	t.Run("success-text", func(t *testing.T) {
		mockUseCase := authMocks.NewMockClientUseCase(t)
		mockUseCase.EXPECT().
			RotateSecret(ctx, clientID).
			Return(&authDomain.CreateClientOutput{
				ID:          clientID,
				PlainSecret: plainSecret,
			}, nil).
			Once()

		var out bytes.Buffer
		err := RunRotateClientSecret(ctx, mockUseCase, logger, &out, clientID.String(), "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Client secret rotated successfully!")
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), plainSecret)
	})

	t.Run("success-json", func(t *testing.T) {
		mockUseCase := authMocks.NewMockClientUseCase(t)
		mockUseCase.EXPECT().
			RotateSecret(ctx, clientID).
			Return(&authDomain.CreateClientOutput{
				ID:          clientID,
				PlainSecret: plainSecret,
			}, nil).
			Once()

		var out bytes.Buffer
		err := RunRotateClientSecret(ctx, mockUseCase, logger, &out, clientID.String(), "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), clientID.String())
		require.Contains(t, out.String(), plainSecret)
	})

	t.Run("invalid-uuid", func(t *testing.T) {
		mockUseCase := authMocks.NewMockClientUseCase(t)
		err := RunRotateClientSecret(ctx, mockUseCase, logger, &bytes.Buffer{}, "invalid-uuid", "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid client ID format")
	})

	t.Run("use-case-error", func(t *testing.T) {
		mockUseCase := authMocks.NewMockClientUseCase(t)
		mockUseCase.EXPECT().
			RotateSecret(ctx, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		err := RunRotateClientSecret(ctx, mockUseCase, logger, &bytes.Buffer{}, clientID.String(), "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to rotate client secret")
	})
}
