package usecase

import (
	"regexp"
	"strings"

	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

var (
	pathRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_/]+$`)
)

func validateSecretPath(path string) error {
	// Check length
	if len(path) < 1 || len(path) > 255 {
		return secretsDomain.ErrInvalidSecretPath
	}

	// Check characters
	if !pathRegex.MatchString(path) {
		return secretsDomain.ErrInvalidSecretPath
	}

	// Check leading/trailing slashes
	if path[0] == '/' || path[len(path)-1] == '/' {
		return secretsDomain.ErrInvalidSecretPath
	}

	// Check consecutive slashes
	if strings.Contains(path, "//") {
		return secretsDomain.ErrInvalidSecretPath
	}

	// Check consecutive symbols
	if strings.Contains(path, "--") || strings.Contains(path, "__") {
		return secretsDomain.ErrInvalidSecretPath
	}

	return nil
}
