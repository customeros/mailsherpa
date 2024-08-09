package syntax

import (
	"regexp"
	"strings"

	"golang.org/x/net/publicsuffix"
)

var (
	usernameRegex = regexp.MustCompile(`^[\p{L}\p{N}.!#$%&'+-/=?^_` + "`" + `{|}~]+$`)
	domainRegex   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9](?:\.[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])*$`)
)

func IsValidEmailSyntax(email string) bool {
	if !isValidEmailFormat(email) {
		return false
	}

	username, domain, ok := splitEmail(email)
	if !ok {
		return false
	}

	return isValidUsername(username) && isValidDomain(domain)
}

func isValidEmailFormat(email string) bool {
	return strings.TrimSpace(email) == email && email != ""
}

func splitEmail(email string) (string, string, bool) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func isValidUsername(username string) bool {
	return len(username) <= 64 &&
		!strings.Contains(username, "*") &&
		usernameRegex.MatchString(username)
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

func GetEmailUserAndDomain(email string) (string, string, bool) {
	if strings.TrimSpace(email) != email {
		return "", "", false
	}
	user, domain, ok := splitEmail(email)
	if !isValidUsername(user) || !isValidDomain(domain) {
		return "", "", false
	}

	return user, domain, ok
}
