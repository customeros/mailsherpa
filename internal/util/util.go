package util

import (
	"strings"
)

// Helper function to check if a string contains any of the given substrings
func ContainsAnyString(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
