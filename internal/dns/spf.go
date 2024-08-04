package dns

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/customeros/mailhawk/internal/syntax"
)

type AuthorizedSenders struct {
	Enterprise    []string
	Finance       []string
	Hosting       []string
	Marketing     []string
	Security      []string
	Support       []string
	Transactional []string
	Webmail       []string
}

func GetAuthorizedSenders(email, knownSPFfilePath string) (AuthorizedSenders, error) {
	knownProviders, err := getKnownProviders(knownSPFfilePath)
	if err != nil {
		return AuthorizedSenders{}, fmt.Errorf("error getting known providers list: %w", err)
	}

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

func processIncludes(spfRecord string, knownProviders Domain) AuthorizedSenders {
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
				senders.Enterprise = append(senders.Enterprise, provider.Name)
			case "finance":
				senders.Finance = append(senders.Finance, provider.Name)
			case "hosting":
				senders.Hosting = append(senders.Hosting, provider.Name)
			case "marketing":
				senders.Marketing = append(senders.Marketing, provider.Name)
			case "security":
				senders.Security = append(senders.Security, provider.Name)
			case "support":
				senders.Support = append(senders.Support, provider.Name)
			case "transactional":
				senders.Transactional = append(senders.Transactional, provider.Name)
			case "webmail":
				senders.Webmail = append(senders.Webmail, provider.Name)
			default:
				log.Printf("'%s' for provider '%s' is unrecognized in known_email_providers.toml", provider.Type, provider.Name)
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
