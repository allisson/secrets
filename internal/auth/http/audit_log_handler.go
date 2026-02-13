// Package http provides HTTP handlers for authentication and client management operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/allisson/secrets/internal/auth/http/dto"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/httputil"
)

// AuditLogHandler handles HTTP requests for audit log operations.
type AuditLogHandler struct {
	auditLogUseCase authUseCase.AuditLogUseCase
	logger          *slog.Logger
}

// NewAuditLogHandler creates a new audit log handler with required dependencies.
func NewAuditLogHandler(
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
) *AuditLogHandler {
	return &AuditLogHandler{
		auditLogUseCase: auditLogUseCase,
		logger:          logger,
	}
}

// ListHandler retrieves audit logs with pagination support.
// GET /v1/audit-logs?offset=0&limit=50 - Requires ReadCapability on path /v1/audit-logs.
// Returns 200 OK with paginated audit log list ordered by ID descending (newest first).
func (h *AuditLogHandler) ListHandler(c *gin.Context) {
	// Parse offset query parameter (default: 0)
	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid offset parameter: must be a non-negative integer"),
			h.logger)
		return
	}

	// Parse limit query parameter (default: 50, max: 100)
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid limit parameter: must be between 1 and 100"),
			h.logger)
		return
	}

	// Call use case
	auditLogs, err := h.auditLogUseCase.List(c.Request.Context(), offset, limit)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Map to response
	response := dto.MapAuditLogsToListResponse(auditLogs)
	c.JSON(http.StatusOK, response)
}
