package syntax

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

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
