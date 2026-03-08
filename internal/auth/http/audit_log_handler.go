// Package http provides HTTP handlers for authentication and client management operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

// ListHandler retrieves audit logs with cursor pagination and optional time-based filtering.
// GET /v1/audit-logs?after_id=<uuid>&limit=50&created_at_from=2026-02-01T00:00:00Z&created_at_to=2026-02-14T23:59:59Z&client_id=<uuid>
// Requires ReadCapability on path /v1/audit-logs. Returns 200 OK with paginated audit log list
// ordered by created_at descending (newest first). Accepts optional created_at_from and
// created_at_to query parameters in RFC3339 format. Timestamps are converted to UTC. Both
// boundaries are inclusive (>= and <=). Uses cursor-based pagination with after_id parameter.
// Accepts optional client_id query parameter (UUID format).
func (h *AuditLogHandler) ListHandler(c *gin.Context) {
	// Parse cursor and limit query parameters
	afterID, limit, err := httputil.ParseUUIDCursorPagination(c, "after_id")
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Parse optional client_id query parameter
	var clientID *uuid.UUID
	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		parsed, err := uuid.Parse(clientIDStr)
		if err != nil {
			httputil.HandleBadRequestGin(c,
				fmt.Errorf("invalid client_id format: must be a valid UUID (e.g., 550e8400-e29b-41d4-a716-446655440000)"),
				h.logger)
			return
		}
		clientID = &parsed
	}

	// Parse optional created_at_from query parameter
	var createdAtFrom *time.Time
	if fromStr := c.Query("created_at_from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			httputil.HandleBadRequestGin(c,
				fmt.Errorf("invalid created_at_from format: must be RFC3339 (e.g., 2026-02-01T00:00:00Z)"),
				h.logger)
			return
		}
		utcTime := parsed.UTC()
		createdAtFrom = &utcTime
	}

	// Parse optional created_at_to query parameter
	var createdAtTo *time.Time
	if toStr := c.Query("created_at_to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			httputil.HandleBadRequestGin(c,
				fmt.Errorf("invalid created_at_to format: must be RFC3339 (e.g., 2026-02-14T23:59:59Z)"),
				h.logger)
			return
		}
		utcTime := parsed.UTC()
		createdAtTo = &utcTime
	}

	// Validate that created_at_from is before or equal to created_at_to
	if createdAtFrom != nil && createdAtTo != nil && createdAtFrom.After(*createdAtTo) {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("created_at_from must be before or equal to created_at_to"),
			h.logger)
		return
	}

	// Call use case with limit + 1 to detect if there are more results
	auditLogs, err := h.auditLogUseCase.ListCursor(
		c.Request.Context(),
		afterID,
		limit+1,
		createdAtFrom,
		createdAtTo,
		clientID,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Determine if there are more results and set next cursor
	var nextCursor *string
	if len(auditLogs) > limit {
		// More results exist, use the last visible item's ID as cursor
		auditLogs = auditLogs[:limit]
		cursorValue := auditLogs[len(auditLogs)-1].ID.String()
		nextCursor = &cursorValue
	}

	// Map to response
	response := dto.MapAuditLogsToListResponse(auditLogs, nextCursor)
	c.JSON(http.StatusOK, response)
}
