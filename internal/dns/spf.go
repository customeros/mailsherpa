package dns

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/customeros/mailsherpa/internal/syntax"
)

type AuthorizedSenders struct {
	Enterprise []string
	Hosting    []string
	Security   []string
	Webmail    []string
	Other      []string
}

func GetAuthorizedSenders(email string, knownProviders *KnownProviders) (AuthorizedSenders, error) {
	spfRecord, err := getSPFRecord(email)
	if err != nil {
		return AuthorizedSenders{}, fmt.Errorf("error getting SPF record: %w", err)
	}
	return processIncludes(spfRecord, knownProviders), nil
}

func getSPFRecord(email string) (string, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return "", fmt.Errorf("invalid email address")
	}
	records, err := net.LookupTXT(domain)
	if err != nil {
		return "", fmt.Errorf("error looking up TXT records: %w", err)
	}
	for _, record := range records {
		if strings.HasPrefix(strings.TrimSpace(record), "v=spf1") {
			return record, nil
		}
	}
	return "", fmt.Errorf("no SPF record found for domain %s", domain)
}

func processIncludes(spfRecord string, knownProviders *KnownProviders) AuthorizedSenders {
	senders := AuthorizedSenders{
		Enterprise: []string{},
		Hosting:    []string{},
		Security:   []string{},
		Webmail:    []string{},
		Other:      []string{},
	}

	includes := regexp.MustCompile(`include:([^\s]+)`).FindAllStringSubmatch(spfRecord, -1)

	categoryMap := map[string]*[]string{
		"enterprise": &senders.Enterprise,
		"hosting":    &senders.Hosting,
		"security":   &senders.Security,
		"webmail":    &senders.Webmail,
		"other":      &senders.Other,
	}

	for _, include := range includes {
		if len(include) < 2 {
			continue
		}
		includeDomain := extractRootDomain(include[1])
		providerName, category := knownProviders.GetProviderByDomain(includeDomain)
		if providerName != "" {
			if slice, exists := categoryMap[category]; exists {
				appendIfNotExists(slice, providerName)
			}
		}
	}

	return senders
}

func appendIfNotExists(slice *[]string, s string) {
	for _, v := range *slice {
		if v == s {
			return
		}
	}
	*slice = append(*slice, s)
}

func extractRootDomain(fullDomain string) string {
	parts := strings.Split(fullDomain, ".")
	if len(parts) <= 2 {
		return fullDomain
	}

	commonTLDs := map[string]bool{
		"com": true, "org": true, "net": true, "edu": true, "gov": true, "co": true,
	}

	tldIndex := len(parts) - 1
	secondLevelDomainIndex := tldIndex - 1

	if commonTLDs[parts[secondLevelDomainIndex]] {
		return strings.Join(parts[secondLevelDomainIndex:], ".")
	}

	return strings.Join(parts[secondLevelDomainIndex:], ".")
}
