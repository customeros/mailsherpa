package mailvalidate

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"strings"

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
	Firewall          string
	AuthorizedSenders dns.AuthorizedSenders
	IsFirewalled      bool
	IsCatchAll        bool
	CanConnectSMTP    bool
}

type EmailValidation struct {
	IsDeliverable   bool
	IsMailboxFull   bool
	IsRoleAccount   bool
	IsFreeAccount   bool
	SmtpSuccess     bool
	ResponseCode    string
	ErrorCode       string
	Description     string
	RetryValidation bool
	SmtpResponse    string
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
	return ValidateDomainWithCustomKnownProviders(validationRequest, *knownProviders, validateCatchAll)
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

	authorizedSenders, err := dns.GetAuthorizedSenders(validationRequest.Email, &knownProviders)
	if err != nil {
		return results, errors.Wrap(err, "Error getting authorized senders from spf records")
	}
	results.AuthorizedSenders = authorizedSenders
	if results.Provider == "unknown" && len(results.AuthorizedSenders.Enterprise) > 0 {
		results.Provider = results.AuthorizedSenders.Enterprise[0]
	}
	if results.Provider == "unknown" && len(results.AuthorizedSenders.Webmail) > 0 {
		results.Provider = results.AuthorizedSenders.Webmail[0]
	}
	if results.Provider == "unknown" && len(results.AuthorizedSenders.Hosting) > 0 {
		results.Provider = results.AuthorizedSenders.Hosting[0]
	}

	if len(results.AuthorizedSenders.Security) > 0 {
		results.IsFirewalled = true
		results.Firewall = results.AuthorizedSenders.Security[0]
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
	results.ResponseCode = smtpValidation.ResponseCode
	results.ErrorCode = smtpValidation.ErrorCode
	results.Description = smtpValidation.Description
	results.SmtpResponse = smtpValidation.SmtpResponse

	switch results.ResponseCode {
	case "250":
		results.SmtpSuccess = true
		results.ErrorCode = ""
		results.Description = ""
	case "251":
		results.SmtpSuccess = true
		results.ErrorCode = ""
		results.Description = ""
	case "450":
		if results.ErrorCode == "4.2.0" {
			results.RetryValidation = true
		}
	case "451":
		if strings.Contains(results.Description, "Internal resources are temporarily unavailable") ||
			strings.Contains(results.Description, "Account service is temporarily unavailable") ||
			strings.Contains(results.Description, "Recipient Temporarily Unavailable") ||
			strings.Contains(results.Description, "IP Temporarily Blacklisted") {
			results.RetryValidation = true
		}
		if results.Description == "Account inbounds disabled" {
			results.SmtpSuccess = true
			results.ErrorCode = ""
			results.Description = ""
		}
		if strings.Contains(results.Description, "Sorry, I wasn’t able to establish an SMTP connection. I’m not going to try again; this message has been in the queue too long.") {
			results.SmtpSuccess = true
			results.ErrorCode = ""
			results.Description = ""
		}
		if results.ErrorCode == "4.7.1" {
			results.RetryValidation = true
		}
	case "501":
		if strings.Contains(results.Description, "Invalid address") {
			results.SmtpSuccess = true
			results.ErrorCode = ""
			results.Description = ""
		}
	case "503":
		if strings.Contains(results.Description, "User unknown") {
			results.SmtpSuccess = true
			results.ErrorCode = ""
			results.Description = ""
		}
	case "550":
		if strings.Contains(results.Description, "Invalid Recipient") || strings.Contains(results.Description, "Recipient not found") {
			results.SmtpSuccess = true
			results.ErrorCode = ""
			results.Description = ""
		}
		if results.ErrorCode == "5.2.1" || results.ErrorCode == "5.7.1" || results.ErrorCode == "5.1.1" || results.ErrorCode == "5.1.6" || results.ErrorCode == "5.1.0" {
			results.SmtpSuccess = true
			results.ErrorCode = ""
			results.Description = ""
		}

	}

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
