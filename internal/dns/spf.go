package dns

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/customeros/mailsherpa/internal/syntax"
)

type AuthorizedSenders struct {
	Enterprise    []string
	Finance       []string
	Hosting       []string
	Marketing     []string
	Sales         []string
	Security      []string
	Support       []string
	Transactional []string
	Webmail       []string
	Other         []string
}

func GetAuthorizedSenders(email string, knownProviders KnownProviders) (AuthorizedSenders, error) {
	spfRecord, err := getSPFRecord(email)
	if err != nil {
		return AuthorizedSenders{}, fmt.Errorf("error getting SPF record: %w", err)
	}

	return processIncludes(spfRecord, knownProviders), nil
}

func getSPFRecord(email string) (string, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return "", fmt.Errorf("Invalid email address")
	}

	records, err := net.LookupTXT(domain)
	if err != nil {
		return "", fmt.Errorf("error looking up TXT records: %v", err)
	}

	for _, record := range records {
		if strings.HasPrefix(strings.TrimSpace(record), "v=spf1") {
			return record, nil
		}
	}

	return "", fmt.Errorf("No SPF record found for domain %s", domain)
}

func processIncludes(spfRecord string, knownProviders KnownProviders) AuthorizedSenders {
	var senders AuthorizedSenders

	includes := regexp.MustCompile(`include:([^\s]+)`).FindAllStringSubmatch(spfRecord, -1)

	for _, include := range includes {
		if len(include) < 2 {
			continue
		}

		includeDomain := extractRootDomain(include[1])
		provider, exists := knownProviders[includeDomain]
		if exists {
			switch provider.Type {
			case "enterprise":
				if !contains(senders.Enterprise, provider.Name) {
					senders.Enterprise = append(senders.Enterprise, provider.Name)
				}
			case "finance":
				if !contains(senders.Finance, provider.Name) {
					senders.Finance = append(senders.Finance, provider.Name)
				}
			case "hosting":
				if !contains(senders.Hosting, provider.Name) {
					senders.Hosting = append(senders.Hosting, provider.Name)
				}
			case "marketing":
				if !contains(senders.Marketing, provider.Name) {
					senders.Marketing = append(senders.Marketing, provider.Name)
				}
			case "sales":
				if !contains(senders.Sales, provider.Name) {
					senders.Sales = append(senders.Sales, provider.Name)
				}
			case "security":
				if !contains(senders.Security, provider.Name) {
					senders.Security = append(senders.Security, provider.Name)
				}
			case "support":
				if !contains(senders.Support, provider.Name) {
					senders.Support = append(senders.Support, provider.Name)
				}
			case "transactional":
				if !contains(senders.Transactional, provider.Name) {
					senders.Transactional = append(senders.Transactional, provider.Name)
				}
			case "webmail":
				if !contains(senders.Webmail, provider.Name) {
					senders.Webmail = append(senders.Webmail, provider.Name)
				}
			case "other":
				if !contains(senders.Other, provider.Name) {
					senders.Other = append(senders.Other, provider.Name)
				}
			default:
				log.Printf("'%s' for provider '%s' is unrecognized, please add to known_email_providers.toml", provider.Type, provider.Name)
			}
		} else {
			log.Printf("'%s' not found in known_email_providers.toml", includeDomain)
		}
	}

	return senders
}

func extractRootDomain(fullDomain string) string {
	// Split the domain by dots
	parts := strings.Split(fullDomain, ".")

	// If we have 2 or fewer parts, return the full domain
	if len(parts) <= 2 {
		return fullDomain
	}

	// Start from the end and look for the first non-common TLD part
	tldIndex := len(parts) - 1
	secondLevelDomainIndex := tldIndex - 1

	// List of common TLDs that might appear as second-level domains
	commonTLDs := map[string]bool{
		"com": true, "org": true, "net": true, "edu": true, "gov": true, "co": true,
	}

	// If the second-level domain is a common TLD, include it
	if commonTLDs[parts[secondLevelDomainIndex]] {
		return strings.Join(parts[secondLevelDomainIndex:], ".")
	}

	// Otherwise, return just the second-level domain and TLD
	return strings.Join(parts[secondLevelDomainIndex:], ".")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
