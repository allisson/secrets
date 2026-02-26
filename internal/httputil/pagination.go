package httputil

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePagination safely parses and validates offset and limit query parameters.
// It uses default values of 0 for offset and 50 for limit.
// The limit cannot exceed 100.
func ParsePagination(c *gin.Context) (offset, limit int, err error) {
	// Parse offset query parameter (default: 0)
	offsetStr := c.DefaultQuery("offset", "0")
	offset, err = strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return 0, 0, fmt.Errorf("invalid offset parameter: must be a non-negative integer")
	}

	// Parse limit query parameter (default: 50, max: 100)
	limitStr := c.DefaultQuery("limit", "50")
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("invalid limit parameter: must be between 1 and 100")
	}

	return offset, limit, nil
}
