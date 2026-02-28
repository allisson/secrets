// Package domain defines core transit encryption domain models.
package domain

const (
	// MaxTransitKeyNameLength is the maximum allowed length for transit key names.
	// This limit aligns with database schema constraints (VARCHAR(255)) and prevents
	// excessively long identifiers that could impact performance or cause display issues.
	MaxTransitKeyNameLength = 255
)
