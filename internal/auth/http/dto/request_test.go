package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

func TestCreateClientRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := CreateClientRequest{
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		req := CreateClientRequest{
			Name:     "",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankName", func(t *testing.T) {
		req := CreateClientRequest{
			Name:     "   ",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_EmptyPolicies", func(t *testing.T) {
		req := CreateClientRequest{
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_InvalidPolicyDocument", func(t *testing.T) {
		req := CreateClientRequest{
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestUpdateClientRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		req := UpdateClientRequest{
			Name:     "Updated Client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/prod/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		req := UpdateClientRequest{
			Name:     "",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_EmptyPolicies", func(t *testing.T) {
		req := UpdateClientRequest{
			Name:     "Updated Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}

func TestValidatePolicyDocument(t *testing.T) {
	t.Run("Success_ValidPolicy", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "/v1/secrets/*",
			Capabilities: []authDomain.Capability{authDomain.ReadCapability, authDomain.WriteCapability},
		}

		err := validatePolicyDocument(policy)
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyPath", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "",
			Capabilities: []authDomain.Capability{authDomain.ReadCapability},
		}

		err := validatePolicyDocument(policy)
		assert.Error(t, err)
	})

	t.Run("Error_BlankPath", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "   ",
			Capabilities: []authDomain.Capability{authDomain.ReadCapability},
		}

		err := validatePolicyDocument(policy)
		assert.Error(t, err)
	})

	t.Run("Error_EmptyCapabilities", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "/v1/secrets/*",
			Capabilities: []authDomain.Capability{},
		}

		err := validatePolicyDocument(policy)
		assert.Error(t, err)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		err := validatePolicyDocument("not a policy")
		assert.Error(t, err)
	})
}

func TestIssueTokenRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		req := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingClientID", func(t *testing.T) {
		req := IssueTokenRequest{
			ClientID:     "",
			ClientSecret: "test_secret_123",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_MissingClientSecret", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		req := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankClientID", func(t *testing.T) {
		req := IssueTokenRequest{
			ClientID:     "   ",
			ClientSecret: "test_secret_123",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankClientSecret", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		req := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "   ",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}
