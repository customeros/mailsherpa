package syntax

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	usernameRegex = regexp.MustCompile(`^[\p{L}\p{N}.!#$%&'+-/=?^_` + "`" + `{|}~]+$`)
	domainRegex   = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
)

func IsValidEmailSyntax(email string) (bool, string) {
	normalizedEmail := convertToAscii(email)

	if !isValidEmailFormat(normalizedEmail) {
		return false, ""
	}

	username, domain, ok := splitEmail(normalizedEmail)
	if !ok {
		return false, ""
	}

	return isValidUsername(username) && isValidDomain(domain), normalizedEmail
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

func convertToAscii(input string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, input)

	// Step 2: Remove any remaining non-ASCII characters
	ascii := make([]rune, 0, len(result))
	for _, r := range result {
		if r <= unicode.MaxASCII {
			ascii = append(ascii, r)
		}
	}

	return string(ascii)
}

func ExtractDomain(fullDomain string) (string, error) {
	var domain string
	if strings.Contains(fullDomain, "://") {
		// It's likely a URL, so parse it
		u, err := url.Parse(fullDomain)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %v", err)
		}
		domain = u.Hostname()
	} else {
		domain = fullDomain
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
