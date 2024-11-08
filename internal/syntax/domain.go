package syntax

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func ExtractRootDomain(fullDomain string) (string, error) {
	domain := fullDomain
	if strings.Contains(fullDomain, "://") {
		// It's likely a URL, so parse it
		u, err := url.Parse(fullDomain)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %v", err)
		}
		domain = u.Hostname()
	}

	// Remove 'www.' prefix if present
	domain = strings.TrimPrefix(domain, "www.")

	// If the domain is already in its simplest form, return it
	if !strings.Contains(domain, ".") ||
		len(strings.Split(domain, ".")) == 2 {
		return domain, nil
	}

	// Get the public suffix (e.g., "co.uk", "com")
	_, icann := publicsuffix.PublicSuffix(domain)
	if !icann {
		// Instead of returning error, just return the domain
		return domain, nil
	}

	// Try to get eTLD+1, if it fails, return original domain
	registrableDomain, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return domain, nil
	}
	return registrableDomain, nil
}

func ParseRootAndSubdomain(input string) (string, string, error) {
	// Ensure the input has a scheme
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}

	input = strings.Replace(input, "http:", "https:", 1)

	// Parse the URL
	u, err := url.Parse(input)
	if err != nil {
		return "", "", err
	}

	// Get the domain and TLD using the public suffix list
	domain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		return "", "", err
	}

	// The subdomain is everything before the domain
	subdomain := strings.TrimSuffix(u.Hostname(), domain)
	subdomain = strings.TrimSuffix(subdomain, ".")

	return domain, subdomain, nil
}

func isValidDomain(domain string) bool {
	// Check if domain starts or ends with a dot
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// Split the domain into its parts
	domainParts := strings.Split(domain, ".")

	// A valid domain must have at least 2 parts
	if len(domainParts) < 2 {
		return false
	}

	// Check each part of the domain
	for _, part := range domainParts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
	}

	// Check if the domain matches the regex pattern
	if !domainRegex.MatchString(domain) {
		fmt.Println("here2")
		return false
	}

	// Extract the TLD using the public suffix list
	tld, _ := publicsuffix.PublicSuffix(domain)
	if tld == "" {
		return false
	}

	// Check if the extracted TLD is valid
	if !isValidTLD(tld) {
		return false
	}

	// Ensure the domain ends with the extracted TLD
	return strings.HasSuffix(domain, "."+tld)
}

func isValidTLD(tld string) bool {
	tld = strings.TrimPrefix(tld, ".")
	_, icann := publicsuffix.PublicSuffix("example." + tld)
	return icann
}
