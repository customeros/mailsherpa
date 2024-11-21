package mailvalidate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/customeros/mailsherpa/mailvalidate"
)

func TestValidateEmailSyntax(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expected    mailvalidate.SyntaxValidation
		description string
	}{
		{
			name:  "Valid Gmail Account",
			email: "john.doe@gmail.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "johndoe",
				Domain:            "gmail.com",
				CleanEmail:        "johndoe@gmail.com",
				IsRoleAccount:     false,
				IsFreeAccount:     true,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Standard Gmail account should be valid and marked as free",
		},
		{
			name:  "Valid Business Email",
			email: "john.doe@microsoft.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "john.doe",
				Domain:            "microsoft.com",
				CleanEmail:        "john.doe@microsoft.com",
				IsRoleAccount:     false,
				IsFreeAccount:     false,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Business email should be valid and not marked as free",
		},
		{
			name:  "Role Account",
			email: "support@company.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "support",
				Domain:            "company.com",
				CleanEmail:        "support@company.com",
				IsRoleAccount:     true,
				IsFreeAccount:     false,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Role account should be identified correctly",
		},
		{
			name:  "System Generated Email",
			email: "draft-ietf-sipcore-refer-explicit-subscription@ietf.org",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "draft-ietf-sipcore-refer-explicit-subscription",
				Domain:            "ietf.org",
				CleanEmail:        "draft-ietf-sipcore-refer-explicit-subscription@ietf.org",
				IsRoleAccount:     true,
				IsFreeAccount:     false,
				IsSystemGenerated: true,
				Error:             "",
			},
			description: "System generated email should be identified correctly",
		},
		{
			name:        "Invalid Email Syntax",
			email:       "not.an.email@",
			expected:    mailvalidate.SyntaxValidation{},
			description: "Invalid email should return empty validation struct",
		},
		{
			name:  "Underscore Username",
			email: "bob_smith@google.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "bob_smith",
				Domain:            "google.com",
				CleanEmail:        "bob_smith@google.com",
				IsRoleAccount:     false,
				IsFreeAccount:     false,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Underscores in username should be supported",
		},
		{
			name:  "Mixed Case with Emoji Email",
			email: "Rob.Name😆@Gmail.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "robname",
				Domain:            "gmail.com",
				CleanEmail:        "robname@gmail.com",
				IsRoleAccount:     false,
				IsFreeAccount:     true,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Mixed case email should be normalized",
		},
		{
			name:  "Email with Plus Addressing",
			email: "bob+tag@yahoo.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "bob+tag",
				Domain:            "yahoo.com",
				CleanEmail:        "bob+tag@yahoo.com",
				IsRoleAccount:     false,
				IsFreeAccount:     true,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Email with plus addressing should be valid",
		},
		{
			name:  "Common Role Account",
			email: "admin@company.com",
			expected: mailvalidate.SyntaxValidation{
				IsValid:           true,
				User:              "admin",
				Domain:            "company.com",
				CleanEmail:        "admin@company.com",
				IsRoleAccount:     true,
				IsFreeAccount:     false,
				IsSystemGenerated: false,
				Error:             "",
			},
			description: "Common role account should be identified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mailvalidate.ValidateEmailSyntax(tt.email)

			// Check each field individually for better error messages
			assert.Equal(t, tt.expected.IsValid, result.IsValid,
				"IsValid mismatch for %s", tt.email)

			if tt.expected.IsValid {
				assert.Equal(t, tt.expected.User, result.User,
					"User mismatch for %s", tt.email)
				assert.Equal(t, tt.expected.Domain, result.Domain,
					"Domain mismatch for %s", tt.email)
				assert.Equal(t, tt.expected.CleanEmail, result.CleanEmail,
					"CleanEmail mismatch for %s", tt.email)
				assert.Equal(t, tt.expected.IsRoleAccount, result.IsRoleAccount,
					"IsRoleAccount mismatch for %s", tt.email)
				assert.Equal(t, tt.expected.IsFreeAccount, result.IsFreeAccount,
					"IsFreeAccount mismatch for %s", tt.email)
				assert.Equal(t, tt.expected.IsSystemGenerated, result.IsSystemGenerated,
					"IsSystemGenerated mismatch for %s", tt.email)
			}
		})
	}
}

// TestEdgeCases tests various edge cases and unusual inputs
func TestEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name  string
		email string
	}{
		{"Empty string", ""},
		{"Single @", "@"},
		{"Multiple @", "user@@domain.com"},
		{"Special characters", "user!#$%@domain.com"},
		{"Very long local part", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz@domain.com"},
		{"Very long domain", "user@abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz.com"},
		{"No TLD", "user@domain"},
		{"Double dots", "user..name@domain.com"},
		{"Leading dot", ".user@domain.com"},
		{"Trailing dot", "user.@domain.com"},
		{"Spaces in email", "user name@domain.com"},
		{"Unicode in email", "üser@domain.com"},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mailvalidate.ValidateEmailSyntax(tc.email)

			// For invalid cases, expect empty validation
			if !result.IsValid {
				assert.Equal(t, mailvalidate.SyntaxValidation{}, result,
					"Expected empty validation for invalid email: %s", tc.email)
			}
		})
	}
}

// TestFreeEmailProviders tests various free email providers
func TestFreeEmailProviders(t *testing.T) {
	freeProviders := []string{
		"gmail.com",
		"yahoo.com",
		"hotmail.com",
		"outlook.com",
		"aol.com",
		"protonmail.com",
		"icloud.com",
	}

	for _, domain := range freeProviders {
		email := "user@" + domain
		t.Run(domain, func(t *testing.T) {
			result := mailvalidate.ValidateEmailSyntax(email)
			assert.True(t, result.IsFreeAccount,
				"Expected %s to be identified as free email provider", domain)
		})
	}
}
