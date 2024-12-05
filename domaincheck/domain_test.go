package domaincheck_test

import (
	"testing"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/stretchr/testify/assert"
)

func TestCheckDNS(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected domaincheck.DNS
	}{
		{
			name:   "Google domain - should have all records",
			domain: "google.com",
			expected: domaincheck.DNS{
				HasA:   true,
				MX:     []string{"smtp.google.com"},
				SPF:    "v=spf1 include:_spf.google.com ~all",
				Errors: []string{},
			},
		},
		{
			name:   "Nonexistent domain",
			domain: "thisisnotarealdomain12345.com",
			expected: domaincheck.DNS{
				HasA: false,
				MX:   []string{},
				SPF:  "",
				Errors: []string{
					"lookup failed: no such host",
				},
			},
		},
		{
			name:   "CustomerOS Docs - should have CNAME",
			domain: "docs.customeros.ai",
			expected: domaincheck.DNS{
				HasA:  true,
				CNAME: "cname.vercel-dns.com",
				Errors: []string{
					"no such host",
				},
			},
		},
		{
			name:   "Domain with no MX records but with A record",
			domain: "cust.cx",
			expected: domaincheck.DNS{
				HasA: true,
				MX:   []string{},
				Errors: []string{
					"no MX records found",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domaincheck.CheckDNS(tt.domain)

			// Basic existence checks rather than exact matches
			if tt.expected.HasA {
				assert.True(t, result.HasA, "Expected A record to exist")
			} else {
				assert.False(t, result.HasA, "Expected no A record")
			}

			// Check if MX records exist when expected
			if len(tt.expected.MX) > 0 {
				assert.NotEmpty(t, result.MX, "Expected MX records to exist")
			}

			// Check if SPF record exists when expected
			if tt.expected.SPF != "" {
				assert.NotEmpty(t, result.SPF, "Expected SPF record to exist")
			}

			// Check if CNAME exists when expected
			if tt.expected.CNAME != "" {
				assert.NotEmpty(t, result.CNAME, "Expected CNAME to exist")
			}

			// Check error cases
			if len(tt.expected.Errors) > 0 {
				assert.NotEmpty(t, result.Errors, "Expected errors to be present")
			} else {
				assert.Empty(t, result.Errors, "Expected no errors")
			}
		})
	}
}

func TestDomainRedirectCheck(t *testing.T) {
	tests := []struct {
		name           string
		domain         string
		expectRedirect bool
		expectedDomain string
		description    string
	}{
		{
			name:           "Domain with no redirect",
			domain:         "google.com",
			expectRedirect: false,
			expectedDomain: "",
			description:    "Google.com should not redirect to a different domain",
		},
		{
			name:           "Domain with HTTPS redirect only",
			domain:         "github.com",
			expectRedirect: false,
			expectedDomain: "",
			description:    "Github.com redirects to HTTPS but stays on same domain",
		},
		{
			name:           "Domain with whitespace",
			domain:         "  openline.ai  ",
			expectRedirect: true,
			expectedDomain: "customeros.ai",
			description:    "Should handle whitespace in input correctly",
		},
		{
			name:           "Non-existent domain",
			domain:         "thisisnotarealdomain12345.com",
			expectRedirect: false,
			expectedDomain: "",
			description:    "Non-existent domain should not show redirects",
		},
		{
			name:           "Youtu.be redirect",
			domain:         "youtu.be",
			expectRedirect: true,
			expectedDomain: "youtube.com",
			description:    "youtu.be should redirect to youtube.com",
		},
		{
			name:           "Openline redirect",
			domain:         "openline.ai",
			expectRedirect: true,
			expectedDomain: "customeros.ai",
			description:    "openline.ai should redirect to customeros.ai",
		},
		{
			name:           "shortened Url",
			domain:         "https://bit.ly/3iPrRWb",
			expectRedirect: true,
			expectedDomain: "freshworks.com",
			description:    "shortened url following redirect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasRedirect, redirectDomain := domaincheck.DomainRedirectCheck(tt.domain)

			if tt.expectRedirect {
				assert.True(t, hasRedirect, "Expected redirect for %s", tt.domain)
				assert.Equal(t, tt.expectedDomain, redirectDomain,
					"Unexpected redirect domain for %s", tt.domain)
			} else {
				assert.False(t, hasRedirect, "Unexpected redirect for %s", tt.domain)
				assert.Empty(t, redirectDomain,
					"Expected empty redirect domain for %s", tt.domain)
			}
		})
	}
}

// TestDomainRedirectCheckTimeout tests the timeout functionality
func TestDomainRedirectCheckTimeout(t *testing.T) {
	// Using a domain that's likely to be slow or timeout
	slowDomain := "example.com:81" // Non-standard port that should timeout

	hasRedirect, redirectDomain := domaincheck.DomainRedirectCheck(slowDomain)

	assert.False(t, hasRedirect, "Should not show redirect for timing out domain")
	assert.Empty(t, redirectDomain, "Should return empty string for timing out domain")
}

func TestPrimaryDomainCheck(t *testing.T) {
	tests := []struct {
		name           string
		domain         string
		isPrimary      bool
		expectedDomain string
		description    string
	}{
		{
			name:           "Google - Primary Domain",
			domain:         "google.com",
			isPrimary:      true,
			expectedDomain: "google.com",
			description:    "Google.com is a primary domain with MX records and A records",
		},
		{
			name:           "Subdomain of Primary",
			domain:         "mail.google.com",
			isPrimary:      false,
			expectedDomain: "google.com",
			description:    "Subdomain should return root domain",
		},
		{
			name:           "GitHub Pages Domain",
			domain:         "username.github.io",
			isPrimary:      false,
			expectedDomain: "",
			description:    "GitHub Pages domains have CNAME and aren't primary",
		},
		{
			name:           "Linktree Exception",
			domain:         "linktr.ee",
			isPrimary:      false,
			expectedDomain: "",
			description:    "Linktree is explicitly excluded",
		},
		{
			name:           "Non-existent Domain",
			domain:         "thisisnotarealdomain12345.com",
			isPrimary:      false,
			expectedDomain: "",
			description:    "Non-existent domains should return false",
		},
		{
			name:           "URL Shortener",
			domain:         "youtu.be",
			isPrimary:      false,
			expectedDomain: "youtube.com",
			description:    "Short URLs should expand to their primary domain",
		},
		{
			name:           "Shopify Store",
			domain:         "store.shopify.com",
			isPrimary:      false,
			expectedDomain: "shopify.com",
			description:    "Shopify subdomain should return root domain",
		},
		{
			name:           "Microsoft - Primary Domain",
			domain:         "microsoft.com",
			isPrimary:      true,
			expectedDomain: "microsoft.com",
			description:    "Microsoft.com is a primary domain",
		},
		{
			name:           "WordPress Site",
			domain:         "example.wordpress.com",
			isPrimary:      false,
			expectedDomain: "wordpress.com",
			description:    "WordPress subdomain should return root domain",
		},
		{
			name:           "Medium Custom Domain",
			domain:         "blog.medium.com",
			isPrimary:      false,
			expectedDomain: "medium.com",
			description:    "Medium subdomain should return root domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPrimary, actualDomain := domaincheck.PrimaryDomainCheck(tt.domain)

			assert.Equal(t, tt.isPrimary, isPrimary,
				"[%s] Expected isPrimary=%v but got %v",
				tt.domain, tt.isPrimary, isPrimary)

			assert.Equal(t, tt.expectedDomain, actualDomain,
				"[%s] Expected domain=%s but got %s",
				tt.domain, tt.expectedDomain, actualDomain)
		})
	}
}

// TestPrimaryDomainCheckWithErrors tests error conditions
func TestPrimaryDomainCheckWithErrors(t *testing.T) {
	errorTests := []struct {
		name        string
		domain      string
		description string
	}{
		{
			name:        "Empty domain",
			domain:      "",
			description: "Empty domain should return false with no primary domain",
		},
		{
			name:        "Invalid domain characters",
			domain:      "domain with spaces.com",
			description: "Invalid domain should return false with no primary domain",
		},
		{
			name:        "Domain with trailing spaces",
			domain:      "  google.com  ",
			description: "Domain with whitespace should be handled correctly",
		},
		{
			name:        "Very long domain",
			domain:      "very-long-subdomain-that-probably-doesnt-exist.very-long-domain-that-probably-doesnt-exist.com",
			description: "Very long domain should be handled gracefully",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			isPrimary, primaryDomain := domaincheck.PrimaryDomainCheck(tt.domain)

			// For error cases, we expect isPrimary to be false
			assert.False(t, isPrimary,
				"Expected isPrimary=false for error case: %s", tt.domain)

			// For error cases, primaryDomain should be empty
			assert.Empty(t, primaryDomain,
				"Expected empty primaryDomain for error case: %s", tt.domain)
		})
	}
}

// TestPrimaryDomainCheckTimeout tests timeout scenarios
func TestPrimaryDomainCheckTimeout(t *testing.T) {
	// Using a domain that's likely to timeout
	slowDomain := "1.2.3.4" // Non-standard port that should timeout

	isPrimary, primaryDomain := domaincheck.PrimaryDomainCheck(slowDomain)

	assert.False(t, isPrimary,
		"Should return false for timing out domain")
	assert.Empty(t, primaryDomain,
		"Should return empty string for timing out domain")
}
