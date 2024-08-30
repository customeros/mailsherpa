package dns

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/customeros/mailsherpa/internal/syntax"
)

type DNS struct {
	MX     []string
	SPF    string
	CNAME  string
	HasA   bool
	Errors []string
}

func GetDNS(email string) DNS {
	var dns DNS
	var mxErr error
	var spfErr error

	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		mxErr = fmt.Errorf("No MX Records:  Invalid email address")
		dns.Errors = append(dns.Errors, mxErr.Error())
		return dns
	}

	dns.HasA = hasAorAAAARecord(domain)

	dns.MX, mxErr = getMXRecordsForDomain(domain)
	dns.SPF, spfErr = getSPFRecord(domain)
	if mxErr != nil {
		dns.Errors = append(dns.Errors, mxErr.Error())
	}
	if spfErr != nil {
		dns.Errors = append(dns.Errors, spfErr.Error())
	}

	exists, cname := getCNAMERecord(domain)
	if exists {
		dns.CNAME = cname
	}
	return dns
}

func getMXRecordsForDomain(domain string) ([]string, error) {
	mxRecords, err := getRawMXRecords(domain)
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

func getRawMXRecords(domain string) ([]*net.MX, error) {
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}

	return mxRecords, nil
}

func getSPFRecord(domain string) (string, error) {
	records, err := net.LookupTXT(domain)
	if err != nil {
		return "", fmt.Errorf("error looking up TXT records: %w", err)
	}
	for _, record := range records {
		spfRecord := parseTXTRecord(record)
		if strings.HasPrefix(spfRecord, "v=spf1") {
			return spfRecord, nil
		}
	}
	return "", fmt.Errorf("no SPF record found for domain %s", domain)
}

func parseTXTRecord(record string) string {
	// Remove surrounding quotes if present
	record = strings.Trim(record, "\"")

	// Replace multiple spaces with a single space
	record = strings.Join(strings.Fields(record), " ")

	return record
}

func getCNAMERecord(domain string) (bool, string) {
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return false, ""
	}

	// Remove the trailing dot from the CNAME if present
	cname = strings.TrimSuffix(cname, ".")

	// Check if the CNAME is different from the input domain
	if cname != domain && cname != domain+"." {
		return true, cname
	}

	return false, ""
}

func hasAorAAAARecord(domain string) bool {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return false
	}
	return len(ips) > 0
}
