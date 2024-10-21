package mailvalidate

import (
	"regexp"
	"strings"
	"unicode"
)

func IsSystemGeneratedUser(user string) bool {
	if isNumeric(user) || isRandomUsername(user) {
		return true
	}
	return false
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
	// Check if the username contains only allowed characters
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9.=_-]+$`)
	if !allowedChars.MatchString(username) {
		return false
	}

	// Check for patterns with many numbers and dashes
	numDashPattern := regexp.MustCompile(`(\d+-){3,}|\d{5,}`)
	if numDashPattern.MatchString(username) {
		return true
	}

	// Check for long hexadecimal-like strings
	hexPattern := regexp.MustCompile(`^[a-f0-9]{10,}$`)
	if hexPattern.MatchString(username) {
		return true
	}

	// Check for multiple segments separated by dots with numbers
	segments := strings.Split(username, ".")
	numericSegments := 0
	for _, segment := range segments {
		if regexp.MustCompile(`^\d+$`).MatchString(segment) {
			numericSegments++
		}
	}
	if numericSegments >= 3 {
		return true
	}

	// Check for long random string followed by a more structured part
	randomStructuredPattern := regexp.MustCompile(`^[a-z0-9]{20,}[-=][a-z0-9._-]+$`)
	if randomStructuredPattern.MatchString(username) {
		return true
	}

	// If none of the above patterns match, it's likely not a random username
	return false
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
