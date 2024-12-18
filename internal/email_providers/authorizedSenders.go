package emailproviders

import (
	"log"
	"regexp"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/syntax"
	"github.com/customeros/mailsherpa/internal/util"
)

type AuthorizedSenders struct {
	Enterprise []string
	Hosting    []string
	Security   []string
	Webmail    []string
	Other      []string
}

func GetAuthorizedSenders(dns domaincheck.DNS, knownProviders *KnownProviders) AuthorizedSenders {
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
		includeDomain, err := syntax.ExtractRootDomain(include[1])
		if err != nil {
			log.Printf("Error: %v", err)
		}
		providerName, category := knownProviders.GetProviderByDomain(includeDomain)
		if providerName != "" {
			if slice, exists := categoryMap[category]; exists {
				util.AppendIfNotExists(slice, providerName)
			}
		}
	}

	return senders
}
