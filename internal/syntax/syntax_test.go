package syntax

import (
	"strings"
	"testing"
)

func TestIsValidEmailSyntax(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"Valid email", "user@example.com", true},
		{"Valid email with numbers", "user123@example.com", true},
		{"Valid email with dots", "user.name@example.com", true},
		{"Valid email with plus", "user+tag@example.com", true},
		{"Valid email with underscore", "user_name@example.com", true},
		{"Valid email with dash in domain", "user@example-domain.com", true},
		{"Valid email with subdomain", "user@subdomain.example.com", true},
		{"Empty string", "", false},
		{"Missing @", "userexample.com", false},
		{"Missing username", "@example.com", false},
		{"Missing domain", "user@", false},
		{"Missing TLD", "user@example", false},
		{"Invalid characters", "user*name@example.com", false},
		{"Multiple @", "user@name@example.com", false},
		{"Trailing dot", "user@example.com.", false},
		{"Leading dot in domain", "user@.example.com", false},
		{"Space in email", "user name@example.com", false},
		{"Trailing space", "user@example.com ", false},
		{"Leading space", " user@example.com", false},
		{"Single character TLD", "user@example.a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidEmailSyntax(tt.email); got != tt.want {
				t.Errorf("IsValidEmailSyntax(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestIsValidEmailFormat(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"Valid format", "user@example.com", true},
		{"Empty string", "", false},
		{"Leading space", " user@example.com", false},
		{"Trailing space", "user@example.com ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidEmailFormat(tt.email); got != tt.want {
				t.Errorf("isValidEmailFormat(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestSplitEmail(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		wantUsername string
		wantDomain   string
		wantOk       bool
	}{
		{"Valid email", "user@example.com", "user", "example.com", true},
		{"No @", "userexample.com", "", "", false},
		{"Multiple @", "user@name@example.com", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsername, gotDomain, gotOk := splitEmail(tt.email)
			if gotUsername != tt.wantUsername || gotDomain != tt.wantDomain || gotOk != tt.wantOk {
				t.Errorf("splitEmail(%q) = %v, %v, %v, want %v, %v, %v",
					tt.email, gotUsername, gotDomain, gotOk, tt.wantUsername, tt.wantDomain, tt.wantOk)
			}
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{"Valid username", "user", true},
		{"Valid with special chars", "user.name+tag_123", true},
		{"Too long", strings.Repeat("a", 65), false},
		{"Contains *", "user*name", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUsername(tt.username); got != tt.want {
				t.Errorf("isValidUsername(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   bool
	}{
		{"Valid domain", "example.com", true},
		{"Valid subdomain", "sub.example.com", true},
		{"Valid with dash", "my-domain.com", true},
		{"Leading dot", ".example.com", false},
		{"Trailing dot", "example.com.", false},
		{"Single label", "localhost", false},
		{"Empty label", "example..com", false},
		{"Label too long", "example." + strings.Repeat("a", 64) + ".com", false},
		{"Single char TLD", "example.a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidDomain(tt.domain); got != tt.want {
				t.Errorf("isValidDomain(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

func TestGetEmailUserAndDomain(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		wantUsername string
		wantDomain   string
		wantOk       bool
	}{
		{"Valid email", "user@example.com", "user", "example.com", true},
		{"Valid email with subdomain", "user@sub.example.com", "user", "sub.example.com", true},
		{"Valid email with plus", "user+tag@example.com", "user+tag", "example.com", true},
		{"Valid email with dots in username", "user.name@example.com", "user.name", "example.com", true},
		{"Valid email with numbers", "user123@example.com", "user123", "example.com", true},
		{"Empty string", "", "", "", false},
		{"No @", "userexample.com", "", "", false},
		{"Multiple @", "user@name@example.com", "", "", false},
		{"Only username", "user@", "", "", false},
		{"Only domain", "@example.com", "", "", false},
		{"Leading space", " user@example.com", "", "", false},
		{"Trailing space", "user@example.com ", "", "", false},
		{"Space in email", "user name@example.com", "", "", false},
		{"Unicode in email", "üser@exämple.com", "üser", "exämple.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsername, gotDomain, gotOk := GetEmailUserAndDomain(tt.email)
			if gotUsername != tt.wantUsername || gotDomain != tt.wantDomain || gotOk != tt.wantOk {
				t.Errorf("GetEmailUserAndDomain(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.email, gotUsername, gotDomain, gotOk, tt.wantUsername, tt.wantDomain, tt.wantOk)
			}
		})
	}
}

func TestIsValidTLD(t *testing.T) {
	tests := []struct {
		name     string
		tld      string
		expected bool
	}{
		{"Valid com TLD", "com", true},
		{"Valid org TLD", "org", true},
		{"Valid net TLD", "net", true},
		{"Valid country code uk", "uk", true},
		{"Valid country code with subdomain co.uk", "co.uk", true},
		{"Valid long TLD education", "education", true},
		{"Invalid TLD", "invalid", false},
		{"Empty string", "", false},
		{"TLD with leading dot", ".com", true},
		{"Invalid TLD with space", "c om", false},
		{"Invalid TLD with special character", "com!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTLD(tt.tld)
			if result != tt.expected {
				t.Errorf("isValidTLD(%q) = %v, want %v", tt.tld, result, tt.expected)
			}
		})
	}
}
