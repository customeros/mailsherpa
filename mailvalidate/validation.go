package mailvalidate

import (
	"fmt"
	"github.com/pkg/errors"
	"log"

	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/mailserver"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type EmailValidationRequest struct {
	Email                string
	FromDomain           string
	FromEmail            string
	GenerateFromEmail    bool
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

type EmailValidation struct {
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

func ValidateDomain(validationRequest EmailValidationRequest, validateCatchAll bool) (DomainValidation, error) {
	knownProviders, err := dns.GetKnownProviders()
	if err != nil {
		log.Fatal(err)
	}
	return ValidateDomainWithCustomKnownProviders(validationRequest, knownProviders, validateCatchAll)
}

func ValidateDomainWithCustomKnownProviders(validationRequest EmailValidationRequest, knownProviders dns.KnownProviders, validateCatchAll bool) (DomainValidation, error) {
	var results DomainValidation
	if err := validateRequest(&validationRequest); err != nil {
		return results, errors.Wrap(err, "Invalid request")
	}

	provider, err := dns.GetEmailProviderFromMx(validationRequest.Email, knownProviders)
	if err != nil {
		return results, errors.Wrap(err, "Error getting provider from MX records")
	}
	results.Provider = provider

	authorizedSenders, err := dns.GetAuthorizedSenders(validationRequest.Email, knownProviders)
	if err != nil {
		return results, errors.Wrap(err, "Error getting authorized senders from spf records")
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

	return results, nil
}

func ValidateEmail(validationRequest EmailValidationRequest) (EmailValidation, error) {
	var results EmailValidation
	if err := validateRequest(&validationRequest); err != nil {
		return results, errors.Wrap(err, "Invalid request")
	}

	freeEmails, err := GetFreeEmailList()
	if err != nil {
		log.Fatal(err)
	}

	roleAccounts, err := GetRoleAccounts()
	if err != nil {
		log.Fatal(err)
	}

	isFreeEmail, err := IsFreeEmailCheck(validationRequest.Email, freeEmails)
	if err != nil {
		return results, errors.Wrap(err, "Error executing free email check")
	}
	results.IsFreeAccount = isFreeEmail

	isRoleAccount, err := IsRoleAccountCheck(validationRequest.Email, roleAccounts)
	if err != nil {
		return results, errors.Wrap(err, "Error executing role account check")
	}
	results.IsRoleAccount = isRoleAccount

	if isRoleAccount && !validationRequest.ValidateRoleAccounts {
		return results, nil
	}

	isVerified, smtpValidation, err := mailserver.VerifyEmailAddress(
		validationRequest.Email,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		validationRequest.Proxy,
	)
	if err != nil {
		return results, errors.Wrap(err, "Error validating email via SMTP")
	}
	results.IsDeliverable = isVerified
	results.IsMailboxFull = smtpValidation.InboxFull
	results.SmtpError = smtpValidation.SMTPError
	return results, nil
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

func validateRequest(request *EmailValidationRequest) error {
	if request.Email == "" {
		return errors.New("Email is required")
	}
	if request.FromDomain == "" {
		return errors.New("FromDomain is required")
	}
	if request.FromEmail == "" {
		firstName, lastName := generateNames()
		request.FromEmail = fmt.Sprintf("%s.%s@%s", firstName, lastName, request.FromDomain)
	}
	if request.CatchAllTestUser == "" {
		request.CatchAllTestUser = generateCatchAllUsername()
	}
	return nil
}
