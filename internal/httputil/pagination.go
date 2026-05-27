package httputil

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// MaxPageLimit is the maximum number of items that can be requested in a single page.
	// Values exceeding this limit will be clamped to this maximum.
	MaxPageLimit = 1000

	// DefaultPageLimit is the default number of items returned per page if not specified.
	DefaultPageLimit = 50
)

// ParseUUIDCursorPagination parses cursor-based pagination parameters for UUID-based cursors.
// It accepts a cursor parameter name (e.g., "after_id") and returns the parsed UUID cursor and limit.
// The cursor is optional (nil if not provided). The limit defaults to 50 and is clamped to 1000.
func ParseUUIDCursorPagination(
	c *gin.Context,
	cursorParam string,
) (afterCursor *uuid.UUID, limit int, err error) {
	// Parse cursor parameter (optional)
	cursorStr := c.Query(cursorParam)
	if cursorStr != "" {
		parsedUUID, err := uuid.Parse(cursorStr)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid %s parameter: must be a valid UUID", cursorParam)
		}
		afterCursor = &parsedUUID
	}

	// Parse limit query parameter (default: 50, max: 1000)
	limitStr := c.DefaultQuery("limit", strconv.Itoa(DefaultPageLimit))
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return nil, 0, fmt.Errorf("invalid limit parameter: must be a positive integer")
	}

	// Clamp limit to maximum allowed value
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}

	return afterCursor, limit, nil
}

// Paginate applies the over-fetch-by-one convention for cursor pagination.
// It calls fetch with limit+1 to detect whether more rows exist; if an extra
// row came back it trims the page to limit and returns the cursor of the last
// visible row (via cursorOf), otherwise next is nil. The whole convention —
// the +1, the trim, and the cursor extraction — lives here so call sites only
// supply the fetch and the cursor field.
func Paginate[T any](
	fetch func(limit int) ([]T, error),
	limit int,
	cursorOf func(T) string,
) (page []T, next *string, err error) {
	rows, err := fetch(limit + 1)
	if err != nil {
		return nil, nil, err
	}

	if len(rows) > limit {
		rows = rows[:limit]
		cursor := cursorOf(rows[len(rows)-1])
		next = &cursor
	}

	return rows, next, nil
}

// ParseStringCursorPagination parses cursor-based pagination parameters for string-based cursors.
// It accepts a cursor parameter name (e.g., "after_path", "after_name") and returns the parsed string cursor and limit.
// The cursor is optional (nil if not provided). The limit defaults to 50 and is clamped to 1000.
func ParseStringCursorPagination(
	c *gin.Context,
	cursorParam string,
) (afterCursor *string, limit int, err error) {
	// Parse cursor parameter (optional)
	cursorStr := c.Query(cursorParam)
	if cursorStr != "" {
		afterCursor = &cursorStr
	}

	// Parse limit query parameter (default: 50, max: 1000)
	limitStr := c.DefaultQuery("limit", strconv.Itoa(DefaultPageLimit))
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return nil, 0, fmt.Errorf("invalid limit parameter: must be a positive integer")
	}

	// Clamp limit to maximum allowed value
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}

	return afterCursor, limit, nil
}
