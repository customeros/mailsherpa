package emailproviders

import (
	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/syntax"
)

func GetEmailProviderFromMx(dns domaincheck.DNS, knownProviders KnownProviders) (emailProvider, firewall string) {
	if len(dns.MX) == 0 {
		return "", ""
	}
	for _, record := range dns.MX {
		domain, err := syntax.ExtractRootDomain(record)
		if err != nil {
			continue
		}
		provider, category := knownProviders.GetProviderByDomain(domain)
		if provider == "" {
			return domain, ""
		}

		switch category {
		case "enterprise", "webmail", "hosting", "security":
			return provider, ""
		default:
			return "", ""
		}
	}

	return "", ""
}
