package mx

import (
	"fmt"
	"net"
	"strings"

	"github.com/customeros/mailhawk/internal/syntax"
)

func getSPFRecords(email string) ([]string, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return []string{}, fmt.Errorf("Invalid email address")
	}

	records, err := net.LookupTXT(domain)
	if err != nil {
		return []string{}, fmt.Errorf("error looking up TXT records: %v", err)
	}
	return records, nil
}

func parseIncludes(spfRecord string) []string {
	var includes []string

	// Split the SPF record into individual terms
	terms := strings.Fields(spfRecord)

	for _, term := range terms {
		if strings.HasPrefix(term, "include:") {
			includes = append(includes, term)
		}
	}

	return includes
}

func GetEmailProvidersFromSPF(email string) ([]string, error) {
	spf, err := getSPFRecords(email)
	if err != nil {
		return []string{}, nil
	}

	providers := []string{}

	for _, record := range spf {
		includes := parseIncludes(record)
		for _, record := range includes {
			exists := EmailProviders[record]
			if exists != "" {
				providers = append(providers, exists)
			}
		}

	}

	return providers, nil
}
