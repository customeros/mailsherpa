package dns

import (
	"fmt"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/syntax"
)

func GetDNS(email string) domaincheck.DNS {
	var dns domaincheck.DNS

	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		mxErr := fmt.Errorf("No MX Records:  Invalid email address")
		dns.Errors = append(dns.Errors, mxErr.Error())
		return dns
	}

	dns = domaincheck.CheckDNS(domain)

	return dns
}
