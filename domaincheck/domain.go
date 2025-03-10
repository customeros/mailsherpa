package domaincheck

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/customeros/mailsherpa/internal/syntax"
	"github.com/miekg/dns"
)

// dnsResolver implements a custom resolver that tries multiple DNS servers
type dnsResolver struct {
	servers []string
	timeout time.Duration
}

// newDNSResolver creates a new DNS resolver with fallback servers
func newDNSResolver() *dnsResolver {
	return &dnsResolver{
		servers: []string{
			"",        // empty string means use system resolver
			"8.8.8.8", // Google DNS
			"1.1.1.1", // Cloudflare DNS
		},
		timeout: 5 * time.Second,
	}
}

// lookupMX performs MX record lookup with fallback to different DNS servers
func (r *dnsResolver) lookupMX(domain string) ([]*net.MX, error) {
	var lastErr error

	for _, server := range r.servers {
		if server == "" {
			// Use system resolver
			mxRecords, err := net.LookupMX(domain)
			if err == nil {
				return mxRecords, nil
			}
			lastErr = err
			continue
		}

		// Use miekg/dns for direct DNS queries
		c := new(dns.Client)
		c.Timeout = r.timeout

		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), dns.TypeMX)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, server+":53")
		if err != nil {
			lastErr = err
			continue
		}

		// Check if domain exists but has no MX records
		if r.Rcode == dns.RcodeSuccess {
			var mxRecords []*net.MX
			for _, ans := range r.Answer {
				if mx, ok := ans.(*dns.MX); ok {
					mxRecords = append(mxRecords, &net.MX{
						Host: strings.TrimSuffix(mx.Mx, "."),
						Pref: mx.Preference,
					})
				}
			}

			// If we got a successful response but no MX records, the domain exists but has no MX records
			if len(mxRecords) == 0 {
				return nil, fmt.Errorf("no MX records found")
			}

			return mxRecords, nil
		}

		lastErr = fmt.Errorf("DNS query failed with code %v", r.Rcode)
	}
	return nil, lastErr
}

// lookupTXT performs TXT record lookup with fallback to different DNS servers
func (r *dnsResolver) lookupTXT(domain string) ([]string, error) {
	var lastErr error
	for _, server := range r.servers {
		if server == "" {
			// Use system resolver
			txtRecords, err := net.LookupTXT(domain)
			if err == nil {
				return txtRecords, nil
			}
			lastErr = err
			continue
		}

		// Use miekg/dns for direct DNS queries
		c := new(dns.Client)
		c.Timeout = r.timeout

		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), dns.TypeTXT)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, server+":53")
		if err != nil {
			lastErr = err
			continue
		}

		if r.Rcode != dns.RcodeSuccess {
			lastErr = fmt.Errorf("DNS query failed with code %v", r.Rcode)
			continue
		}

		var txtRecords []string
		for _, ans := range r.Answer {
			if txt, ok := ans.(*dns.TXT); ok {
				txtRecords = append(txtRecords, strings.Join(txt.Txt, ""))
			}
		}

		if len(txtRecords) > 0 {
			return txtRecords, nil
		}
	}
	return nil, lastErr
}

// lookupCNAME performs CNAME record lookup with fallback to different DNS servers
func (r *dnsResolver) lookupCNAME(domain string) (string, error) {
	var lastErr error
	for _, server := range r.servers {
		if server == "" {
			// Use system resolver
			cname, err := net.LookupCNAME(domain)
			if err == nil {
				return cname, nil
			}
			lastErr = err
			continue
		}

		// Use miekg/dns for direct DNS queries
		c := new(dns.Client)
		c.Timeout = r.timeout

		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), dns.TypeCNAME)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, server+":53")
		if err != nil {
			lastErr = err
			continue
		}

		if r.Rcode != dns.RcodeSuccess {
			lastErr = fmt.Errorf("DNS query failed with code %v", r.Rcode)
			continue
		}

		for _, ans := range r.Answer {
			if cname, ok := ans.(*dns.CNAME); ok {
				return strings.TrimSuffix(cname.Target, "."), nil
			}
		}
	}
	return "", lastErr
}

// lookupIP performs IP lookup with fallback to different DNS servers
func (r *dnsResolver) lookupIP(domain string) ([]net.IP, error) {
	var lastErr error
	for _, server := range r.servers {
		if server == "" {
			// Use system resolver
			ips, err := net.LookupIP(domain)
			if err == nil {
				return ips, nil
			}
			lastErr = err
			continue
		}

		// Use miekg/dns for direct DNS queries
		c := new(dns.Client)
		c.Timeout = r.timeout

		var ips []net.IP

		// Try A records
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, server+":53")
		if err == nil && r.Rcode == dns.RcodeSuccess {
			for _, ans := range r.Answer {
				if a, ok := ans.(*dns.A); ok {
					ips = append(ips, a.A)
				}
			}
		}

		// Try AAAA records
		m = new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), dns.TypeAAAA)
		m.RecursionDesired = true

		r, _, err = c.Exchange(m, server+":53")
		if err == nil && r.Rcode == dns.RcodeSuccess {
			for _, ans := range r.Answer {
				if aaaa, ok := ans.(*dns.AAAA); ok {
					ips = append(ips, aaaa.AAAA)
				}
			}
		}

		if len(ips) > 0 {
			return ips, nil
		}

		if err != nil {
			lastErr = err
		}
	}
	return nil, lastErr
}

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
	if mxErr != nil {
		dns.Errors = append(dns.Errors, mxErr.Error())
	}

	dns.SPF, spfErr = getSPFRecord(domain)
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
	resolver := newDNSResolver()
	return resolver.lookupMX(domain)
}

func getSPFRecord(domain string) (string, error) {
	resolver := newDNSResolver()
	records, err := resolver.lookupTXT(domain)
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
	resolver := newDNSResolver()
	cname, err := resolver.lookupCNAME(domain)
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
	resolver := newDNSResolver()
	ips, err := resolver.lookupIP(domain)
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
