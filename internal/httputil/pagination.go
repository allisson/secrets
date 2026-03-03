package httputil

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// MaxPageLimit is the maximum number of items that can be requested in a single page.
	// Values exceeding this limit will be clamped to this maximum.
	MaxPageLimit = 1000
)

// ParsePagination safely parses and validates offset and limit query parameters.
// It uses default values of 0 for offset and 50 for limit.
// The limit is clamped to a maximum of 1000 if a higher value is requested.
func ParsePagination(c *gin.Context) (offset, limit int, err error) {
	// Parse offset query parameter (default: 0)
	offsetStr := c.DefaultQuery("offset", "0")
	offset, err = strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return 0, 0, fmt.Errorf("invalid offset parameter: must be a non-negative integer")
	}

	// Parse limit query parameter (default: 50, max: 1000)
	limitStr := c.DefaultQuery("limit", "50")
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return 0, 0, fmt.Errorf("invalid limit parameter: must be a positive integer")
	}

	// Clamp limit to maximum allowed value
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}

	return offset, limit, nil
}
