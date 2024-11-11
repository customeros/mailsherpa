package syntax

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func NormalizeEmailAddress(email string) (ok bool, cleanEmail, cleanUser, cleanDomain string) {
	normalizedEmail := convertToAscii(email)

	if !isValidEmailFormat(normalizedEmail) {
		return false, "", "", ""
	}

	username, domain, ok := parseUserAndDomain(normalizedEmail)
	if !ok {
		return false, "", "", ""
	}

	isValid := isValidUsername(username) && isValidDomain(domain)

	if domain == "gmail.com" || domain == "googlemail.com" {
		username = strings.ReplaceAll(username, ".", "")
		domain = "gmail.com"
		normalizedEmail = fmt.Sprintf("%s@%s", username, domain)
	}

	return isValid, normalizedEmail, username, domain
}

func isValidEmailFormat(email string) bool {
	return strings.TrimSpace(email) == email && email != ""
}

func parseUserAndDomain(email string) (string, string, bool) {
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

	return strings.ToLower(string(ascii))
}
