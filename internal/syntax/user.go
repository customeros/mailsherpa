package syntax

import (
	"regexp"
	"strings"

	"github.com/customeros/mailsherpa/internal/util"
)

func IsSystemGeneratedUser(username string) bool {
	if isCommonNamePattern(username) {
		return false
	}

	if util.IsNumericString(username) ||
		isCommonSystemGeneratedPattern(username) ||
		util.IsHighEntropyString(username) ||
		isRandomAfterHyphen(username) ||
		hasMulipleNumericSegments(username) {
		return true
	}

	return false
}

func isCommonNamePattern(username string) bool {
	namePattern := regexp.MustCompile(`^[a-z]+\.[a-z]+\d{1,4}$`)
	if namePattern.MatchString(username) && len(username) < 20 {
		return true
	}
	return false
}

func isCommonSystemGeneratedPattern(username string) bool {
	systemPatterns := []*regexp.Regexp{
		// ld- and usr- patterns
		regexp.MustCompile(`^(ld|usr)-[a-z0-9]{8,}$`),
		// Unsubscribe patterns
		regexp.MustCompile(`^unsub-[a-f0-9]{8}`),
		regexp.MustCompile(`^[0-9]+\.[a-z0-9]{30,}`),
		// UUID patterns
		regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`),
		// System email prefixes
		regexp.MustCompile(`^(bounce|return|system|noreply|no-reply|donotreply|do-not-reply|unsubscribe)[-.][a-z0-9]`),
		// Common prefixes with random strings
		regexp.MustCompile(`^(usr|user|tmp|temp|random)-[a-z0-9]{8,}$`),
	}

	for _, pattern := range systemPatterns {
		if pattern.MatchString(username) {
			return true
		}
	}

	// Quick checks for obvious patterns
	if strings.Count(username, "_") > 2 ||
		strings.Contains(username, "=") ||
		strings.Contains(username, "--") ||
		len(username) >= 40 {
		return true
	}
	return false
}

func isRandomAfterHyphen(username string) bool {
	parts := strings.Split(username, "-")
	if len(parts) == 2 && len(parts[1]) >= 8 {
		return util.IsHighEntropyString(parts[1])
	}
	return false
}

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

func hasMulipleNumericSegments(username string) bool {
	segments := strings.Split(username, ".")
	numericSegments := 0
	for _, segment := range segments {
		if regexp.MustCompile(`^\d+$`).MatchString(segment) {
			numericSegments++
		}
	}

	return numericSegments >= 3
}
