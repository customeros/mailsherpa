package mailvalidate

import (
	"fmt"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/email_providers"
	"github.com/customeros/mailsherpa/internal/free_emails"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type DomainValidationParams struct {
	IsPrimaryDomain bool
	PrimaryDomain   string
}

// DomainValidation contains the complete domain validation results
type DomainValidation struct {
	// Provider information
	Provider              string
	SecureGatewayProvider string
	AuthorizedSenders     emailproviders.AuthorizedSenders

	// Domain flags
	IsFirewalled    bool
	IsCatchAll      bool
	IsPrimaryDomain bool
	HasMXRecord     bool
	HasSPFRecord    bool

	// Domain details
	PrimaryDomain string

	// Server responses
	SmtpResponse     SmtpResponse
	MailServerHealth MailServerHealth

	// Error information
	Error string
}

// ValidateDomain performs complete domain validation for an email
func ValidateDomain(validationRequest EmailValidationRequest) DomainValidation {
	knownProviders, err := emailproviders.GetKnownProviders()
	if err != nil {
		return DomainValidation{
			Error: fmt.Sprintf("Error getting known providers: %v", err),
		}
	}
	return validateDomainWithKnownProviders(validationRequest, *knownProviders)
}

// validateDomainWithKnownProviders performs the actual domain validation
func validateDomainWithKnownProviders(validationRequest EmailValidationRequest, knownProviders emailproviders.KnownProviders) DomainValidation {
	results := DomainValidation{}

	// Validate request
	if err := validateRequest(&validationRequest); err != nil {
		results.Error = fmt.Sprintf("Invalid request: %v", err)
		return results
	}

	ok, _, _, domain := syntax.NormalizeEmailAddress(validationRequest.Email)
	if !ok {
		results.Error = "Invalid email address"
		return results
	}

	// Ensure DNS records are available
	if validationRequest.Dns == nil {
		dns := domaincheck.CheckDNS(domain)
		validationRequest.Dns = &dns
	}

	// Evaluate DNS records and get provider information
	evaluateDnsRecords(&validationRequest, &knownProviders, &results)

	// Check if it's a primary domain
	results.IsPrimaryDomain, results.PrimaryDomain = domaincheck.PrimaryDomainCheck(domain)

	// Check for free email
	isFreeEmail, err := freemail.IsFreeEmailCheck(domain)
	if err != nil {
		results.Error = fmt.Sprintf("Error running free email check: %v", err)
		return results
	}

	// Only perform catch-all test for non-free email domains
	if !isFreeEmail {
		if catchAllResults := catchAllTest(&validationRequest); catchAllResults.IsDeliverable == "true" {
			results.IsCatchAll = true
			results.MailServerHealth = catchAllResults.MailServerHealth
			results.SmtpResponse = catchAllResults.SmtpResponse
		}
	}

	return results
}

// evaluateDnsRecords analyzes DNS records to determine email provider and security settings
func evaluateDnsRecords(validationRequest *EmailValidationRequest, knownProviders *emailproviders.KnownProviders, results *DomainValidation) {
	// Check MX records
	if len(validationRequest.Dns.MX) > 0 {
		results.HasMXRecord = true
		if provider, firewall := emailproviders.GetEmailProviderFromMx(*validationRequest.Dns, *knownProviders); provider != "" {
			results.Provider = provider
			if firewall != "" {
				results.SecureGatewayProvider = firewall
				results.IsFirewalled = true
			}
		}
	}

	// Check SPF records
	if validationRequest.Dns.SPF != "" {
		results.HasSPFRecord = true
		results.AuthorizedSenders = emailproviders.GetAuthorizedSenders(*validationRequest.Dns, knownProviders)
	}

	// Set provider based on authorized senders if not already set
	if results.Provider == "" {
		results.Provider = determineProvider(results.AuthorizedSenders)
	}

	// Check for firewall if not already determined
	if !results.IsFirewalled && len(results.AuthorizedSenders.Security) > 0 {
		results.IsFirewalled = true
		results.SecureGatewayProvider = results.AuthorizedSenders.Security[0]
	}
}

// determineProvider selects the most appropriate provider from authorized senders
func determineProvider(senders emailproviders.AuthorizedSenders) string {
	if len(senders.Enterprise) > 0 {
		return senders.Enterprise[0]
	}
	if len(senders.Webmail) > 0 {
		return senders.Webmail[0]
	}
	if len(senders.Hosting) > 0 {
		return senders.Hosting[0]
	}
	return ""
}
