package domaincheck

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/customeros/mailsherpa/internal/syntax"
)

type DNS struct {
	MX     []string
	SPF    string
	CNAME  string
	HasA   bool
	Errors []string
}

func PrimaryDomainCheck(domain string) (bool, string) {
	root, subdomain, err := parseRootAndSubdomain(domain)
	if err != nil {
		root = domain
	}
	dns := CheckDNS(root)
	redirects, primaryDomain := CheckRedirects(root)

	if !redirects && dns.CNAME == "" && len(dns.MX) > 0 && dns.HasA {
		if subdomain == "" {
			return true, ""
		} else {
			return false, root
		}
	}

	return false, primaryDomain
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

func CheckRedirects(domain string) (bool, string) {
	// Check for HTTP/HTTPS redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	for _, protocol := range []string{"http", "https"} {
		url := fmt.Sprintf("%s://%s", protocol, domain)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			if location != "" {
				location, _ = syntax.ExtractDomain(location)
				if location != domain {
					return true, location
				}
			}
		}
	}

	return false, ""
}

func parseRootAndSubdomain(input string) (string, string, error) {
	// Ensure the input has a scheme
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}

	// Parse the URL
	u, err := url.Parse(input)
	if err != nil {
		return "", "", err
	}

	// Get the domain and TLD using the public suffix list
	domain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		return "", "", err
	}

	// The subdomain is everything before the domain
	subdomain := strings.TrimSuffix(u.Hostname(), domain)
	subdomain = strings.TrimSuffix(subdomain, ".")

	return domain, subdomain, nil
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
