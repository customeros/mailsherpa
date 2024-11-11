package syntax_test

import (
	"testing"

	"github.com/customeros/mailsherpa/internal/syntax"
	"github.com/stretchr/testify/assert"
)

func TestIsSystemGeneratedUser(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		expected    bool
		description string
	}{
		// System Generated Patterns - Should return true
		{
			name:        "LD Pattern",
			username:    "ld-79aaeb1250",
			expected:    true,
			description: "LD prefix with random string",
		},
		{
			name:        "Unsubscribe Pattern",
			username:    "unsub-4077d96e",
			expected:    true,
			description: "Unsubscribe pattern with hex",
		},
		{
			name:        "Numeric with Long String",
			username:    "32.mrtvirzriftuiqkppdxshsysccr",
			expected:    true,
			description: "Number prefix with long random string",
		},
		{
			name:        "Multiple Underscores",
			username:    "user_123_abc_456",
			expected:    true,
			description: "Multiple underscores in username",
		},
		{
			name:        "Double Hyphen",
			username:    "user--123abc",
			expected:    true,
			description: "Username with double hyphen",
		},
		{
			name:        "Very Long Username",
			username:    "thisusernameisveryverylongandshouldbeflaggedassystemgenerated",
			expected:    true,
			description: "Username longer than 40 characters",
		},
		{
			name:        "Multiple Numeric Segments",
			username:    "123.456.789",
			expected:    true,
			description: "Three numeric segments",
		},
		{
			name:        "High Entropy String",
			username:    "x7k9m2p4v8n1",
			expected:    true,
			description: "Random looking string",
		},

		// Non-System Generated Patterns - Should return false
		{
			name:        "Common Name Pattern",
			username:    "john.doe123",
			expected:    false,
			description: "Common firstname.lastname pattern",
		},
		{
			name:        "Simple Username",
			username:    "jsmith",
			expected:    false,
			description: "Simple username without special characters",
		},
		{
			name:        "Name with Single Number",
			username:    "john.doe1",
			expected:    false,
			description: "Name with single digit",
		},
		{
			name:        "Short Name with Hyphen",
			username:    "mary-jane",
			expected:    false,
			description: "Regular hyphenated name",
		},
		{
			name:        "Common Initial Pattern",
			username:    "jdoe",
			expected:    false,
			description: "Common initial plus lastname",
		},
		{
			name:        "Common Initial Pattern with periods",
			username:    "j.d.foe",
			expected:    false,
			description: "Common initial plus lastname",
		},
		{
			name:        "Gmail break pattern",
			username:    "john.m.doe",
			expected:    false,
			description: "Common initial plus lastname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syntax.IsSystemGeneratedUser(tt.username)
			assert.Equal(t, tt.expected, result,
				"[%s] Expected IsSystemGenerated=%v but got %v for username: %s",
				tt.name, tt.expected, result, tt.username)
		})
	}
}

// TestEdgeCases tests boundary conditions
func TestSystemGeneratedUserEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		username string
		expected bool
	}{
		{"Empty String", "", false},
		{"Single Character", "a", false},
		{"Single Number", "1", true},
		{"Equal Sign", "user=123", true},
		{"UUID Format", "550e8400-e29b-41d4-a716-446655440000", true},
		{"Long Numbers", "123456789", true},
		{"Dots Only", "...", false},
		{"System Prefix", "system.1234", true},
		//{"Bounce Prefix", "bounce.user123", true},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := syntax.IsSystemGeneratedUser(tc.username)
			assert.Equal(t, tc.expected, result,
				"Edge case [%s] expected %v but got %v for username: %s",
				tc.name, tc.expected, result, tc.username)
		})
	}
}

// TestSpecificSystemPatterns tests each system pattern separately
func TestSpecificSystemPatterns(t *testing.T) {
	systemPatterns := []struct {
		name     string
		username string
		expected bool
	}{
		{"LD Pattern", "ld-12345678", true},
		//{"USR Pattern", "usr-12345678", true},
		{"Unsubscribe Pattern", "unsub-abcd1234", true},
		{"Numeric Dot Pattern", "123.abcdefghijklmnopqrstuvwxyzabcdef", true},
		{"UUID Pattern", "550e8400-e29b-41d4-a716-446655440000", true},
		{"Bounce Pattern", "bounce.12345", true},
		{"Return Pattern", "return.12345", true},
		{"System Pattern", "system.12345", true},
		{"No Reply Pattern", "noreply.12345", true},
		{"No-Reply Pattern", "no-reply-wz2igrh6xmefeptmzkhu7a", true},
		{"Do Not Reply Pattern", "donotreply.12345", true},
		{"Random Prefix Pattern", "random-12345678", true},
		//{"Short Random", "fu5au9", true},
		{"Phone Number", "+14132193236", true},
		{"Number in Middle", "lorenzo422sandoval", false},
	}

	for _, tp := range systemPatterns {
		t.Run(tp.name, func(t *testing.T) {
			result := syntax.IsSystemGeneratedUser(tp.username)
			assert.Equal(t, tp.expected, result,
				"System pattern [%s] expected %v but got %v for username: %s",
				tp.name, tp.expected, result, tp.username)
		})
	}
}
