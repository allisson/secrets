package commands_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/allisson/secrets/cmd/app/commands"
	"github.com/allisson/secrets/internal/keyring"
)

func TestRunRewrapDeks_InvalidKekID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fake := keyring.NewFake()

	err := commands.RunRewrapDeks(context.Background(), fake, logger, "not-a-uuid", 100)
	assert.ErrorContains(t, err, "invalid kek-id")
}

func TestRunRewrapDeks_InvalidBatchSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fake := keyring.NewFake()

	err := commands.RunRewrapDeks(
		context.Background(),
		fake,
		logger,
		uuid.Nil.String(), // Fake's ActiveKekID() returns Nil
		0,
	)
	assert.ErrorContains(t, err, "batch-size must be greater than 0")
}

func TestRunRewrapDeks_MismatchedActiveKek(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fake := keyring.NewFake()

	err := commands.RunRewrapDeks(
		context.Background(),
		fake,
		logger,
		uuid.New().String(), // doesn't match Fake's Nil active id
		100,
	)
	assert.ErrorContains(t, err, "does not match keyring active KEK")
}

func TestRunRewrapDeks_SuccessNoDEKs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fake := keyring.NewFake()

	err := commands.RunRewrapDeks(
		context.Background(),
		fake,
		logger,
		uuid.Nil.String(),
		100,
	)
	assert.NoError(t, err)
}
