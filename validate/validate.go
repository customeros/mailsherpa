package validate

import (
	"fmt"
	"log"

	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/mailserver"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type EmailValidationRequest struct {
	Email                string
	FromDomain           string
	FromEmail            string
	CatchAllTestUser     string
	ValidateFreeAccounts bool
	ValidateRoleAccounts bool
	Proxy                mailserver.ProxySetup
}

type DomainValidation struct {
	Provider          string
	AuthorizedSenders dns.AuthorizedSenders
	IsFirewalled      bool
	IsCatchAll        bool
	CanConnectSMTP    bool
}

type EmailValidatation struct {
	IsDeliverable bool
	IsMailboxFull bool
	IsRoleAccount bool
	IsFreeAccount bool
	SmtpError     string
}

type SyntaxValidation struct {
	IsValid bool
	User    string
	Domain  string
}

func ValidateEmailSyntax(email string) SyntaxValidation {
	var results SyntaxValidation
	ok := syntax.IsValidEmailSyntax(email)
	if !ok {
		return results
	}

	user, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return results
	}
	results.IsValid = true
	results.User = user
	results.Domain = domain
	return results
}

func ValidateDomain(validationRequest EmailValidationRequest, knownProviders dns.KnownProviders, validateCatchAll bool) DomainValidation {
	var results DomainValidation

	provider, err := dns.GetEmailProviderFromMx(validationRequest.Email, knownProviders)
	if err != nil {
		log.Println("Error getting provider from MX records: %w", err)
	}
	results.Provider = provider

	authorizedSenders, err := dns.GetAuthorizedSenders(validationRequest.Email, knownProviders)
	if err != nil {
		log.Println("Error getting authorized senders from spf records: %w", err)
	}
	results.AuthorizedSenders = authorizedSenders
	if results.Provider == "" && len(results.AuthorizedSenders.Enterprise) > 0 {
		results.Provider = results.AuthorizedSenders.Enterprise[0]
	}
	if results.Provider == "" && len(results.AuthorizedSenders.Webmail) > 0 {
		results.Provider = results.AuthorizedSenders.Webmail[0]
	}

	if len(results.AuthorizedSenders.Security) > 0 {
		results.IsFirewalled = true
	}

	if validateCatchAll {
		var smptResults mailserver.SMPTValidation
		results.IsCatchAll, smptResults = catchAllTest(validationRequest)
		results.CanConnectSMTP = smptResults.CanConnectSmtp

	}

	return results
}

func ValidateEmail(validationRequest EmailValidationRequest, knownProviders dns.KnownProviders, freeEmails FreeEmails, roleAccounts RoleAccounts) EmailValidatation {
	var results EmailValidatation

	isFreeEmail, err := IsFreeEmailCheck(validationRequest.Email, freeEmails)
	if err != nil {
		log.Printf("Error executing free email check: %v", err)
	}
	results.IsFreeAccount = isFreeEmail

	isRoleAccount, err := IsRoleAccountCheck(validationRequest.Email, roleAccounts)
	if err != nil {
		log.Printf("Error executing role account check: %v", err)
	}
	results.IsRoleAccount = isRoleAccount

	if isRoleAccount && !validationRequest.ValidateRoleAccounts {
		return results
	}

	isVerified, smtpValidation, err := mailserver.VerifyEmailAddress(
		validationRequest.Email,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		validationRequest.Proxy,
	)
	if err != nil {
		log.Printf("Error validating email via SMTP: %v", err)
	}
	results.IsDeliverable = isVerified
	results.IsMailboxFull = smtpValidation.InboxFull
	results.SmtpError = smtpValidation.SMTPError
	return results
}

func catchAllTest(validationRequest EmailValidationRequest) (bool, mailserver.SMPTValidation) {
	_, domain, ok := syntax.GetEmailUserAndDomain(validationRequest.Email)
	if !ok {
		log.Printf("Invalid email address")
		return false, mailserver.SMPTValidation{}
	}

	catchAllEmail := fmt.Sprintf("%s@%s", validationRequest.CatchAllTestUser, domain)
	isVerified, smtpValidation, err := mailserver.VerifyEmailAddress(
		catchAllEmail,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		validationRequest.Proxy,
	)
	if err != nil {
		log.Printf("Error validating email via SMTP: %v", err)
		return false, mailserver.SMPTValidation{}
	}

	return isVerified, smtpValidation
}
