package syntax

import (
	"regexp"
	"strings"

	"github.com/customeros/mailsherpa/internal/util"
)

// IsSystemGeneratedUser checks if a given username is system-generated
func IsSystemGeneratedUser(username string) bool {
	if username == "" {
		return false
	}

	// Exclude human-like names
	if isCommonNamePattern(username) || isHyphenatedName(username) ||
		isWithinName(username) {
		return false
	}

	// Direct pattern match for system-generated usernames
	if containsSystemGeneratedKeyword(username) || matchesSystemGeneratedPattern(username) {
		return true
	}

	// Check for usernames with high entropy or complex numeric patterns
	if util.IsHighEntropyString(username) || util.IsNumericString(username) ||
		hasMultipleNumericSegments(username) ||
		isPhoneNumber(username) {
		return true
	}

	return false
}

// Detect phone numbers
func isPhoneNumber(username string) bool {
	phonePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\+?\d{10,15}$`),             // International format
		regexp.MustCompile(`^\(\d{3}\)\s?\d{3}-?\d{4}$`), // (123) 456-7890
		regexp.MustCompile(`^\d{3}-?\d{3}-?\d{4}$`),      // 123-456-7890
	}

	for _, pattern := range phonePatterns {
		if pattern.MatchString(username) {
			return true
		}
	}
	return false
}

// Check if a number is within a natural name
func isWithinName(username string) bool {
	// Check if the username follows a natural name pattern with numbers
	naturalNamePattern := regexp.MustCompile(`^[a-zA-Z]+\d{1,3}[a-zA-Z]+$`)
	return naturalNamePattern.MatchString(username)
}

// containsSystemGeneratedKeyword checks for common keywords in system-generated usernames
func containsSystemGeneratedKeyword(username string) bool {
	username = strings.ToLower(username)

	// Check for keywords in the username
	keywords := []string{
		"usr-", "noreply", "bounce", "system", "return", "unsub-", "ld-", "do-not-reply", "donotreply",
	}

	for _, keyword := range keywords {
		if strings.Contains(username, keyword) {
			return true
		}
	}
	return false
}

// matchesSystemGeneratedPattern checks for specific patterns indicating system-generated usernames
func matchesSystemGeneratedPattern(username string) bool {
	username = strings.ToLower(username)

	patterns := []*regexp.Regexp{
		// General system prefixes with alphanumeric suffixes
		regexp.MustCompile(`^(usr|ld)-[a-z0-9]+$`),
		regexp.MustCompile(`^(noreply|bounce|system|return)[._-]?[a-z0-9]+$`),

		// UUID pattern
		regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`),

		// Specific unsubscribe pattern
		regexp.MustCompile(`^unsub-[a-f0-9]{8}$`),
	}

	for _, pattern := range patterns {
		if pattern.MatchString(username) {
			return true
		}
	}

	// Additional criteria for complex usernames that suggest system generation
	if len(username) >= 40 || strings.Count(username, "_") > 2 || strings.Contains(username, "=") {
		return true
	}

	return false
}

// hasMultipleNumericSegments checks for usernames with multiple numeric segments
func hasMultipleNumericSegments(username string) bool {
	segments := strings.Split(username, ".")
	numericSegments := 0
	for _, segment := range segments {
		if regexp.MustCompile(`^\d+$`).MatchString(segment) {
			numericSegments++
		}
	}
	return numericSegments >= 3
}

// isCommonNamePattern checks if the username matches typical human naming patterns
func isCommonNamePattern(username string) bool {
	namePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[a-z]+\.[a-z]+\d{0,4}$`),  // john.doe123
		regexp.MustCompile(`^[a-z]+\_[a-z]+\d{0,4}$`),  // john.doe123
		regexp.MustCompile(`^[a-z]+\.[a-z]\.[a-z]+$`),  // john.m.doe
		regexp.MustCompile(`^[a-z]+\.[a-z]+\.[a-z]+$`), // john.michael.doe
		regexp.MustCompile(`^[a-z]+[a-z0-9]{0,4}$`),    // johndoe, john123
	}

	username = strings.ToLower(username)
	for _, pattern := range namePatterns {
		if pattern.MatchString(username) && len(username) < 30 {
			return true
		}
	}
	return false
}

// isHyphenatedName checks if the username resembles a hyphenated name
func isHyphenatedName(username string) bool {
	namePattern := regexp.MustCompile(`^[a-z]+-[a-z]+$`)
	return namePattern.MatchString(strings.ToLower(username))
}

// isValidUsername checks basic validation for allowed characters in email usernames
func isValidUsername(username string) bool {
	if len(username) > 64 {
		return false
	}

	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9.=_+!#$%&'*+/=?^_{|}~-]+$`)
	return allowedChars.MatchString(username)
}
