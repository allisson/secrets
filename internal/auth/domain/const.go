// Package domain defines authentication and authorization domain models.
// Implements capability-based access control with clients, tokens, policies, and audit logging.
package domain

// Capability defines the types of operations that can be performed on resources.
// Capabilities are used in policy documents to control client authorization.
type Capability string

const (
	// ReadCapability allows reading resource data.
	ReadCapability Capability = "read"

	// WriteCapability allows creating or updating resource data.
	WriteCapability Capability = "write"

	// DeleteCapability allows removing resource data.
	DeleteCapability Capability = "delete"

	// EncryptCapability allows encrypting data using cryptographic keys.
	EncryptCapability Capability = "encrypt"

	// DecryptCapability allows decrypting data using cryptographic keys.
	DecryptCapability Capability = "decrypt"

	// RotateCapability allows rotating cryptographic keys.
	RotateCapability Capability = "rotate"
)
