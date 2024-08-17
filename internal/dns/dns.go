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
	Errors []string
}

func GetDNS(email string) DNS {
	var dns DNS
	var mxErr error
	var spfErr error

	dns.MX, mxErr = getMXRecordsForEmail(email)
	dns.SPF, spfErr = getSPFRecord(email)
	if mxErr != nil {
		dns.Errors = append(dns.Errors, mxErr.Error())
	}
	if spfErr != nil {
		dns.Errors = append(dns.Errors, spfErr.Error())
	}
	return dns
}

func getMXRecordsForEmail(email string) ([]string, error) {
	mxRecords, err := getRawMXRecords(email)
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

func getRawMXRecords(email string) ([]*net.MX, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return nil, fmt.Errorf("Invalid domain")
	}

	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}

	return mxRecords, nil
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
