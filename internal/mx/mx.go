package mx

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/customeros/mailhawk/internal/syntax"
)

func getRawMXRecords(email string) ([]*net.MX, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return nil, fmt.Errorf("Invalid domain")
	}

	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}

	return mxRecords, nil
}

func GetMXRecordsForEmail(email string) ([]string, error) {
	mxRecords, err := getRawMXRecords(email)
	if err != nil {
		return nil, err
	}

	// Sort MX records by priority (lower number = higher priority)
	sort.Slice(mxRecords, func(i, j int) bool {
		return mxRecords[i].Pref < mxRecords[j].Pref
	})

	stripDot := func(s string) string {
		return strings.ToLower(strings.TrimSuffix(s, "."))
	}

	// Extract hostnames into a string array
	result := make([]string, len(mxRecords))
	for i, mx := range mxRecords {
		result[i] = stripDot(mx.Host)
	}

	return result, nil
}

func GetEmailServiceProviderFromMX(mxRecords []string) string {
	if len(mxRecords) == 0 {
		return ""
	}

	// Use the first MX record as a reference
	parts := strings.Split(mxRecords[0], ".")
	numParts := len(parts)

	if numParts < 2 {
		return ""
	}

	// Start with the last two parts as the potential root domain
	potentialRoot := strings.Join(parts[numParts-2:], ".")

	// Check if all MX records contain this potential root
	for _, record := range mxRecords {
		if !strings.HasSuffix(record, potentialRoot) {
			// If not, return the last part only (TLD)
			return parts[numParts-1]
		}
	}

	return potentialRoot
}
