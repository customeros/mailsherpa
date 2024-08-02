package syntax

import (
	"regexp"
	"strings"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'+-/=?^_` + "`" + `{|}~]+$`)
	domainRegex   = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
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
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	domainParts := strings.Split(domain, ".")
	if len(domainParts) < 2 {
		return false
	}

	for _, part := range domainParts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
	}

	return len(domainParts[len(domainParts)-1]) >= 2 &&
		domainRegex.MatchString(domain)
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

