// Package mocks provides mock implementations for testing HTTP handlers.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// MockTokenUseCase is a mock implementation of TokenUseCase for testing.
type MockTokenUseCase struct {
	mock.Mock
}

// Issue mocks the Issue method of TokenUseCase.
func (m *MockTokenUseCase) Issue(
	ctx context.Context,
	input *authDomain.IssueTokenInput,
) (*authDomain.IssueTokenOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.IssueTokenOutput), args.Error(1)
}

// Authenticate mocks the Authenticate method of TokenUseCase.
func (m *MockTokenUseCase) Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.Client), args.Error(1)
}
