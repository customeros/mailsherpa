package dns

func GetEmailProviderFromMx(dns DNS, knownProviders KnownProviders) (emailProvider, firewall string) {
	if len(dns.MX) == 0 {
		return "", ""
	}
	for _, record := range dns.MX {
		domain := extractRootDomain(record)
		provider, category := knownProviders.GetProviderByDomain(domain)
		if provider == "" {
			return domain, ""
		}

		switch category {
		case "enterprise":
			return provider, ""
		case "webmail":
			return provider, ""
		case "hosting":
			return provider, ""
		case "security":
			return "", provider
		default:
			return "", ""
		}
	}

	return "", ""
}
