package domaincheck

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/customeros/mailsherpa/internal/syntax"
)

type DNS struct {
	MX     []string
	SPF    string
	CNAME  string
	HasA   bool
	Errors []string
}

func CheckDNS(domain string) DNS {
	var dns DNS
	var mxErr, spfErr error

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

func DomainRedirectCheck(domain string) (bool, string) {
	domain = cleanDomain(domain)

	// Initialize final redirect location
	var finalLoc string

	// Configure HTTP client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}

	// Check both HTTP and HTTPS
	for _, protocol := range []string{"http", "https"} {
		url := fmt.Sprintf("%s://%s", protocol, domain)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Check if it's a redirect status code (300-399)
		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			continue
		}

		location := resp.Header.Get("Location")
		if location == "" || strings.HasPrefix(location, "/") {
			continue
		}

		// Extract root domain from redirect location
		redirectDomain, err := syntax.ExtractRootDomain(location)
		if err != nil {
			continue
		}

		// If redirect domain is different from original domain
		if redirectDomain != domain {
			finalLoc = redirectDomain
			return true, finalLoc
		}
	}

	// No valid redirects found
	return false, ""
}

func PrimaryDomainCheck(domain string) (bool, string) {
	var expanded bool
	domain, expanded = expandShortURL(domain)

	domain = cleanDomain(domain)

	// Parse domain into root and subdomain
	root, subdomain, err := syntax.ParseRootAndSubdomain(domain)
	if err != nil {
		root = domain
	}

	// Exclude known exceptions
	if root == "linktr.ee" {
		return false, ""
	}

	// Check if domain is accessible
	if !checkConnection(root) {
		return false, ""
	}

	// Check for redirects
	hasRedirect, primaryDomain := DomainRedirectCheck(root)

	// Get DNS information
	dnsInfo := CheckDNS(root)

	// Check if domain is a primary domain
	isPrimaryDomain := !hasRedirect &&
		dnsInfo.CNAME == "" &&
		len(dnsInfo.MX) > 0 &&
		dnsInfo.HasA

	if isPrimaryDomain {
		// If no subdomain and domain wasn't expanded from a shortener,
		// it's a valid primary domain
		if subdomain == "" && !expanded {
			return true, domain
		}
		// Otherwise, return the root domain
		return false, root
	}

	return false, primaryDomain
}

func cleanDomain(domain string) string {

	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.Trim(domain, "/")
	domain = strings.TrimSpace(domain)
	return domain
}

func checkConnection(domain string) bool {
	// Try both HTTP and HTTPS ports
	for _, port := range []string{":80", ":443"} {
		conn, err := net.DialTimeout("tcp", domain+port, time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
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

func parseTXTRecord(record string) string {
	// Remove surrounding quotes if present
	record = strings.Trim(record, "\"")

	// Replace multiple spaces with a single space
	record = strings.Join(strings.Fields(record), " ")

	return record
}

func expandShortURL(domain string) (string, bool) {
	urlShorteners := []string{
		"bit.ly/",
		"hubs.ly/",
	}

	for _, shortener := range urlShorteners {
		if strings.Contains(domain, shortener) {
			isRedirect, expandedDomain := DomainRedirectCheck(domain)
			if isRedirect {
				return expandedDomain, true
			}
			break
		}
	}
	return domain, false
}
