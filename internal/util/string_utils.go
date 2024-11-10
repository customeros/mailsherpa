package util

import (
	"regexp"
	"strings"
	"unicode"
)

func AppendIfNotExists(slice *[]string, s string) {
	for _, v := range *slice {
		if v == s {
			return
		}
	}
	*slice = append(*slice, s)
}

// Helper function to check if a string contains any of the given substrings
func ContainsAnyString(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func IsNumericString(s string) bool {
	for _, char := range s {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

func IsHighEntropyString(s string) bool {
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
