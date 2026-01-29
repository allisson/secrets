package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/allisson/go-project-template/internal/httputil"
	userDomain "github.com/allisson/go-project-template/internal/user/domain"
	userHttp "github.com/allisson/go-project-template/internal/user/http"
	"github.com/allisson/go-project-template/internal/user/http/dto"
	userUsecase "github.com/allisson/go-project-template/internal/user/usecase"
)

// MockUserUseCase is a mock implementation of usecase.UserUseCase
type MockUserUseCase struct {
	mock.Mock
}

func (m *MockUserUseCase) RegisterUser(
	ctx context.Context,
	input userUsecase.RegisterUserInput,
) (*userDomain.User, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserUseCase) GetUserByEmail(ctx context.Context, email string) (*userDomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserUseCase) GetUserByID(ctx context.Context, id uuid.UUID) (*userDomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func TestMakeJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		body         interface{}
		statusCode   int
		expectedBody string
	}{
		{
			name:         "success response",
			body:         map[string]string{"status": "ok"},
			statusCode:   http.StatusOK,
			expectedBody: `{"status":"ok"}`,
		},
		{
			name:         "error response",
			body:         map[string]string{"error": "something went wrong"},
			statusCode:   http.StatusInternalServerError,
			expectedBody: `{"error":"something went wrong"}`,
		},
		{
			name: "complex object",
			body: map[string]interface{}{
				"id":   1,
				"name": "Test",
				"data": map[string]string{"key": "value"},
			},
			statusCode:   http.StatusOK,
			expectedBody: `{"data":{"key":"value"},"id":1,"name":"Test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			httputil.MakeJSONResponse(w, tt.statusCode, tt.body)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestHealthHandler(t *testing.T) {
	handler := HealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestReadinessHandler_Ready(t *testing.T) {
	ctx := context.Background()
	handler := ReadinessHandler(ctx)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
}

func TestReadinessHandler_NotReady(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context to simulate shutdown

	handler := ReadinessHandler(ctx)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not ready", response["status"])
}

func TestLoggingMiddleware(t *testing.T) {
	// Create a test logger
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	// Create a simple handler that returns 200 OK
	simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) //nolint:errcheck,gosec
	})

	// Wrap with logging middleware
	wrapped := LoggingMiddleware(logger)(simpleHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestRecoveryMiddleware(t *testing.T) {
	// Create a test logger
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	wrapped := RecoveryMiddleware(logger)(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", response["error"])
}

func TestChainMiddleware(t *testing.T) {
	// Create middleware that adds headers
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-1", "value1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-2", "value2")
			next.ServeHTTP(w, r)
		})
	}

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain middlewares
	chained := ChainMiddleware(middleware1, middleware2)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "value1", w.Header().Get("X-Test-1"))
	assert.Equal(t, "value2", w.Header().Get("X-Test-2"))
}

func TestUserHandler_Register_Success(t *testing.T) {
	mockUseCase := &MockUserUseCase{}
	handler := userHttp.NewUserHandler(mockUseCase, nil)

	req := dto.RegisterUserRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecurePass123!",
	}

	input := userUsecase.RegisterUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	uuid1 := uuid.Must(uuid.NewV7())
	expectedUser := &userDomain.User{
		ID:    uuid1,
		Name:  input.Name,
		Email: input.Email,
	}

	mockUseCase.On("RegisterUser", mock.Anything, input).Return(expectedUser, nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RegisterUser(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, uuid1.String(), response["id"])
	assert.Equal(t, input.Name, response["name"])
	assert.Equal(t, input.Email, response["email"])

	mockUseCase.AssertExpectations(t)
}

func TestUserHandler_Register_InvalidJSON(t *testing.T) {
	mockUseCase := &MockUserUseCase{}
	handler := userHttp.NewUserHandler(mockUseCase, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RegisterUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "validation_error", response["error"])
}

func TestUserHandler_Register_ValidationError(t *testing.T) {
	mockUseCase := &MockUserUseCase{}
	handler := userHttp.NewUserHandler(mockUseCase, nil)

	tests := []struct {
		name  string
		input dto.RegisterUserRequest
	}{
		{
			name: "empty name",
			input: dto.RegisterUserRequest{
				Name:     "",
				Email:    "john@example.com",
				Password: "SecurePass123!",
			},
		},
		{
			name: "empty email",
			input: dto.RegisterUserRequest{
				Name:     "John Doe",
				Email:    "",
				Password: "SecurePass123!",
			},
		},
		{
			name: "empty password",
			input: dto.RegisterUserRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.RegisterUser(w, req)

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "invalid_input", response["error"])
			assert.Contains(t, response["message"], "required")
		})
	}
}

func TestUserHandler_Register_UseCaseError(t *testing.T) {
	mockUseCase := &MockUserUseCase{}
	handler := userHttp.NewUserHandler(mockUseCase, nil)

	req := dto.RegisterUserRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecurePass123!",
	}

	input := userUsecase.RegisterUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	useCaseError := errors.New("database error")
	mockUseCase.On("RegisterUser", mock.Anything, input).Return(nil, useCaseError)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RegisterUser(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "internal_error", response["error"])

	mockUseCase.AssertExpectations(t)
}

func TestUserHandler_Register_MethodNotAllowed(t *testing.T) {
	mockUseCase := &MockUserUseCase{}
	handler := userHttp.NewUserHandler(mockUseCase, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()

	handler.RegisterUser(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
