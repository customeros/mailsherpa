package dns

import (
	"regexp"
	"strings"
)

type AuthorizedSenders struct {
	Enterprise []string
	Hosting    []string
	Security   []string
	Webmail    []string
	Other      []string
}

func GetAuthorizedSenders(dns DNS, knownProviders *KnownProviders) AuthorizedSenders {
	if dns.SPF == "" {
		return AuthorizedSenders{}
	}
	return processIncludes(dns.SPF, knownProviders)
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
