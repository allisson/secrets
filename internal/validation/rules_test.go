package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordStrength(t *testing.T) {
	rule := PasswordStrength{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
	}

	tests := []struct {
		name      string
		password  string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid password",
			password:  "SecurePass123!",
			shouldErr: false,
		},
		{
			name:      "too short",
			password:  "Short1!",
			shouldErr: true,
			errMsg:    "password must be at least",
		},
		{
			name:      "missing uppercase",
			password:  "securepass123!",
			shouldErr: true,
			errMsg:    "uppercase letter",
		},
		{
			name:      "missing lowercase",
			password:  "SECUREPASS123!",
			shouldErr: true,
			errMsg:    "lowercase letter",
		},
		{
			name:      "missing number",
			password:  "SecurePass!",
			shouldErr: true,
			errMsg:    "number",
		},
		{
			name:      "missing special char",
			password:  "SecurePass123",
			shouldErr: true,
			errMsg:    "special character",
		},
		{
			name:      "all requirements met with symbols",
			password:  "MyP@ssw0rd!",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.password)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordStrength_CustomRequirements(t *testing.T) {
	// Test with only minimum length requirement
	rule := PasswordStrength{
		MinLength:      10,
		RequireUpper:   false,
		RequireLower:   false,
		RequireNumber:  false,
		RequireSpecial: false,
	}

	tests := []struct {
		name      string
		password  string
		shouldErr bool
	}{
		{
			name:      "meets minimum length",
			password:  "tencharact",
			shouldErr: false,
		},
		{
			name:      "below minimum length",
			password:  "short",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.password)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		shouldErr bool
	}{
		{
			name:      "valid email",
			email:     "user@example.com",
			shouldErr: false,
		},
		{
			name:      "valid email with subdomain",
			email:     "user@mail.example.com",
			shouldErr: false,
		},
		{
			name:      "valid email with plus",
			email:     "user+tag@example.com",
			shouldErr: false,
		},
		{
			name:      "valid email with dots",
			email:     "first.last@example.com",
			shouldErr: false,
		},
		{
			name:      "invalid - no @",
			email:     "userexample.com",
			shouldErr: true,
		},
		{
			name:      "invalid - no domain",
			email:     "user@",
			shouldErr: true,
		},
		{
			name:      "invalid - no local part",
			email:     "@example.com",
			shouldErr: true,
		},
		{
			name:      "invalid - no TLD",
			email:     "user@example",
			shouldErr: true,
		},
		{
			name:      "invalid - spaces",
			email:     "user @example.com",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email.Validate(tt.email)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoWhitespace(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "no whitespace",
			input:     "validstring",
			shouldErr: false,
		},
		{
			name:      "leading whitespace",
			input:     " validstring",
			shouldErr: true,
		},
		{
			name:      "trailing whitespace",
			input:     "validstring ",
			shouldErr: true,
		},
		{
			name:      "both leading and trailing",
			input:     " validstring ",
			shouldErr: true,
		},
		{
			name:      "internal spaces allowed",
			input:     "valid string",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NoWhitespace.Validate(tt.input)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotBlank(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "valid string",
			input:     "validstring",
			shouldErr: false,
		},
		{
			name:      "only spaces",
			input:     "   ",
			shouldErr: true,
		},
		{
			name:      "only tabs",
			input:     "\t\t",
			shouldErr: true,
		},
		{
			name:      "only newlines",
			input:     "\n\n",
			shouldErr: true,
		},
		{
			name:      "mixed whitespace",
			input:     " \t\n ",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NotBlank.Validate(tt.input)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWrapValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error returns nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "wraps validation error",
			err:      assert.AnError,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapValidationError(tt.err)
			if tt.expected {
				assert.Error(t, result)
				assert.Contains(t, result.Error(), "invalid input")
			} else {
				assert.NoError(t, result)
			}
		})
	}
}
