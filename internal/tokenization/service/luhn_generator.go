package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

type luhnGenerator struct{}

// NewLuhnGenerator creates a new Luhn algorithm compliant token generator. Generates
// cryptographically secure random numeric tokens that pass Luhn validation (used for payment cards).
func NewLuhnGenerator() TokenGenerator {
	return &luhnGenerator{}
}

// Generate creates a Luhn algorithm compliant numeric token of the specified length.
// The last digit is calculated as the Luhn check digit. Returns an error if length is less than 2.
func (g *luhnGenerator) Generate(length int) (string, error) {
	if length < 2 {
		return "", errors.New("length must be at least 2 for Luhn tokens")
	}
	if length > 255 {
		return "", errors.New("length must not exceed 255")
	}

	// Generate random digits for all positions except the last one
	digits := make([]int, length)
	for i := 0; i < length-1; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate random digit: %w", err)
		}
		digits[i] = int(n.Int64())
	}

	// Calculate and append the Luhn check digit
	digits[length-1] = calculateLuhnCheckDigit(digits[:length-1])

	// Convert to string
	token := make([]byte, length)
	for i, d := range digits {
		token[i] = byte('0' + d)
	}

	return string(token), nil
}

// Validate checks if the token is Luhn algorithm compliant.
func (g *luhnGenerator) Validate(token string) error {
	if len(token) < 2 {
		return errors.New("token must be at least 2 characters for Luhn validation")
	}

	// Check if all characters are numeric
	digits := make([]int, len(token))
	for i, c := range token {
		if c < '0' || c > '9' {
			return errors.New("token must contain only numeric characters")
		}
		digits[i] = int(c - '0')
	}

	// Validate using Luhn algorithm
	if !validateLuhn(digits) {
		return errors.New("token failed Luhn validation")
	}

	return nil
}

// calculateLuhnCheckDigit calculates the Luhn check digit for the given digits.
// The digits slice should NOT include the check digit position.
func calculateLuhnCheckDigit(digits []int) int {
	sum := 0
	length := len(digits)

	// Process digits from right to left (excluding the check digit position)
	for i := 0; i < length; i++ {
		digit := digits[length-1-i]

		// Double every second digit from the right
		if i%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	// Calculate check digit
	checkDigit := (10 - (sum % 10)) % 10
	return checkDigit
}

// validateLuhn validates a complete number (including check digit) using the Luhn algorithm.
func validateLuhn(digits []int) bool {
	sum := 0
	length := len(digits)

	// Process all digits from right to left
	for i := 0; i < length; i++ {
		digit := digits[length-1-i]

		// Double every second digit from the right (skipping the check digit itself)
		if i%2 == 1 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return sum%10 == 0
}
