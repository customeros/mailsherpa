package syntax

import (
	"regexp"
)

func isValidUsername(username string) bool {
	if len(username) > 64 {
		return false
	}

	// Basic validation for allowed characters in email usernames
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9.=_+!#$%&'*+/=?^_{|}~-]+$`)
	if !allowedChars.MatchString(username) {
		return false
	}

	return true
}
