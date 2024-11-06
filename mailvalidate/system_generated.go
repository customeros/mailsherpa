package mailvalidate

import (
	"regexp"
	"strings"
	"unicode"
)

// IsSystemGeneratedUser checks if a username appears to be system generated
func IsSystemGeneratedUser(user string) bool {
	return isNumeric(user) || isRandomUsername(user)
}

func isNumeric(s string) bool {
	for _, char := range s {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

func isRandomUsername(username string) bool {
	// Basic validation for allowed characters in email usernames
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9.=_+!#$%&'*+/=?^_{|}~-]+$`)
	if !allowedChars.MatchString(username) {
		return false
	}

	// Skip common name patterns (initials/name with numbers)
	namePattern := regexp.MustCompile(`^[a-z]+\.[a-z]+\d{1,4}$`)
	if namePattern.MatchString(username) && len(username) < 20 {
		return false
	}

	// Common system-generated patterns - check these before entropy
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

	// Check entropy for non-name-like patterns
	if isHighEntropy(username) && len(username) > 12 {
		return true
	}

	// Quick checks for obvious patterns
	if strings.Count(username, "_") > 2 ||
		strings.Contains(username, "=") ||
		strings.Contains(username, "--") ||
		len(username) >= 40 {
		return true
	}

	// Check if string after hyphen is random-looking
	parts := strings.Split(username, "-")
	if len(parts) == 2 && len(parts[1]) >= 8 {
		return isHighEntropy(parts[1])
	}

	// Check for multiple numeric segments
	segments := strings.Split(username, ".")
	numericSegments := 0
	for _, segment := range segments {
		if regexp.MustCompile(`^\d+$`).MatchString(segment) {
			numericSegments++
		}
	}

	return numericSegments >= 3
}

func isHighEntropy(s string) bool {
	if len(s) < 8 {
		return false
	}

	// Skip if it looks like a name with numbers but ensure we don't skip system patterns
	nameWithNumbersPattern := regexp.MustCompile(`^[a-z]+\.?[a-z]*\d{1,4}$`)
	if nameWithNumbersPattern.MatchString(s) && len(s) < 20 {
		return false
	}

	charMap := make(map[rune]bool)
	consecutiveSame := 1
	maxConsecutive := 1
	var lastChar rune

	for i, char := range s {
		charMap[char] = true

		if i > 0 {
			if char == lastChar {
				consecutiveSame++
				if consecutiveSame > maxConsecutive {
					maxConsecutive = consecutiveSame
				}
			} else {
				consecutiveSame = 1
			}
		}
		lastChar = char
	}

	uniqueRatio := float64(len(charMap)) / float64(len(s))
	consecutiveRatio := float64(maxConsecutive) / float64(len(s))
	transitions := countTransitions(s)

	return ((uniqueRatio > 0.6 && len(s) > 12) ||
		(uniqueRatio > 0.7 && len(s) >= 8) ||
		transitions > 4) &&
		consecutiveRatio < 0.3
}

func countTransitions(s string) int {
	transitions := 0
	prevType := otherType
	for _, char := range s {
		currentType := charTypeCheck(char)
		if currentType != prevType && prevType != otherType && currentType != otherType {
			transitions++
		}
		prevType = currentType
	}
	return transitions
}

type charType int

const (
	otherType charType = iota
	letterType
	digitType
)

func charTypeCheck(r rune) charType {
	if unicode.IsLetter(r) {
		return letterType
	} else if unicode.IsDigit(r) {
		return digitType
	}
	return otherType
}

