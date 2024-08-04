package dns

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strings"

	"github.com/customeros/mailhawk/internal/syntax"
)

func GetEmailProviderFromMx(email string, knownProviders KnownProviders) (string, error) {
	mx, err := GetMXRecordsForEmail(email)
	if err != nil {
		return "", err
	}

	for _, record := range mx {
		domain := extractRootDomain(record)
		provider, exists := knownProviders[domain]
		if !exists {
			log.Printf("Email provider unknown, please add `%s` to known_email_providers.toml", record)
		}

		if provider.Type == "enterprise" {
			return provider.Name, nil
		}
	}

	return "", nil
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

func getEmailServiceProviderFromMX(mxRecords []string) string {
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
	root := strings.Join(parts[numParts-2:], ".")

	// Check if all MX records contain this potential root
	for _, record := range mxRecords {
		if !strings.HasSuffix(record, root) {
			// If not, return the last part only (TLD)
			return parts[numParts-1]
		}
	}

	return root
}

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
