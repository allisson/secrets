package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSecretPath(t *testing.T) {
	// Re-defining tests more carefully
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Success cases
		{"ValidSimple", "app-v1_secret", true},
		{"ValidWithSlash", "prod/database/password", true},
		{"ValidHyphenUnderscore", "my-app_test/secret", true},
		{"ValidNumbers", "app123/v2/key", true},
		{"ValidSingleChar", "a", true},
	}
	// Fixing max length test case
	longPath := ""
	for i := 0; i < 255; i++ {
		longPath += "a"
	}
	tests = append(tests, struct {
		name     string
		path     string
		expected bool
	}{"ValidMaxLen", longPath, true})

	// Failure cases
	failureTests := []struct {
		name string
		path string
	}{
		{"Empty", ""},
		{"TooLong", longPath + "a"},
		{"LeadingSlash", "/app/api-key"},
		{"TrailingSlash", "app/api-key/"},
		{"ConsecutiveSlashes", "app//api-key"},
		{"ConsecutiveHyphens", "app--api-key"},
		{"ConsecutiveUnderscores", "app__api-key"},
		{"InvalidChar_Dot", "app.api-key"},
		{"InvalidChar_Space", "app api-key"},
		{"InvalidChar_At", "app@api-key"},
		{"InvalidChar_Unicode", "app/api-key-🔐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretPath(tt.path)
			if tt.expected {
				assert.NoError(t, err, "Path '%s' should be valid", tt.path)
			} else {
				assert.Error(t, err, "Path '%s' should be invalid", tt.path)
				assert.Equal(t, "invalid secret path format: invalid input", err.Error())
			}
		})
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretPath(tt.path)
			assert.Error(t, err, "Path '%s' should be invalid", tt.path)
			assert.Equal(t, "invalid secret path format: invalid input", err.Error())
		})
	}
}
