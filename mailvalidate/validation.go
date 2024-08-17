package mailvalidate

import (
	"fmt"
	"log"
	"strings"

	"github.com/pkg/errors"

	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/mailserver"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type EmailValidationRequest struct {
	Email            string
	FromDomain       string
	FromEmail        string
	CatchAllTestUser string
	Dns              dns.DNS
}

type DomainValidation struct {
	Provider          string
	Firewall          string
	AuthorizedSenders dns.AuthorizedSenders
	IsFirewalled      bool
	IsCatchAll        bool
	CanConnectSMTP    bool
	HasMXRecord       bool
	HasSPFRecord      bool
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

	if len(validationRequest.Dns.MX) != 0 {
		results.HasMXRecord = true
		provider, firewall := dns.GetEmailProviderFromMx(validationRequest.Dns, knownProviders)
		results.Provider = provider
		if firewall != "" {
			results.Firewall = firewall
			results.IsFirewalled = true
		}
	}

	if validationRequest.Dns.SPF != "" {
		results.HasSPFRecord = true
		authorizedSenders := dns.GetAuthorizedSenders(validationRequest.Dns, &knownProviders)
		results.AuthorizedSenders = authorizedSenders
	}

	if results.Provider == "" && len(results.AuthorizedSenders.Enterprise) > 0 {
		results.Provider = results.AuthorizedSenders.Enterprise[0]
	}
	if results.Provider == "" && len(results.AuthorizedSenders.Webmail) > 0 {
		results.Provider = results.AuthorizedSenders.Webmail[0]
	}
	if results.Provider == "" && len(results.AuthorizedSenders.Hosting) > 0 {
		results.Provider = results.AuthorizedSenders.Hosting[0]
	}

	if !results.IsFirewalled && len(results.AuthorizedSenders.Security) > 0 {
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

	isVerified, smtpValidation, err := mailserver.VerifyEmailAddress(
		validationRequest.Email,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		validationRequest.Dns,
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

	finalResults := handleSmtpResponses(results)

	return finalResults, nil
}

func handleSmtpResponses(resp EmailValidation) EmailValidation {
	switch resp.ResponseCode {
	case "250":
		resp.SmtpSuccess = true
		resp.ErrorCode = ""
		resp.Description = ""
	case "251":
		resp.SmtpSuccess = true
		resp.ErrorCode = ""
		resp.Description = ""
	case "450":
		if resp.ErrorCode == "4.2.0" {
			resp.RetryValidation = true
		}
	case "451":
		if strings.Contains(resp.Description, "Internal resources are temporarily unavailable") ||
			strings.Contains(resp.Description, "Account service is temporarily unavailable") ||
			strings.Contains(resp.Description, "Recipient Temporarily Unavailable") ||
			strings.Contains(resp.Description, "IP Temporarily Blacklisted") {
			resp.RetryValidation = true
		}
		if resp.Description == "Account inbounds disabled" {
			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}
		if strings.Contains(resp.Description, "Sorry, I wasn’t able to establish an SMTP connection. I’m not going to try again; this message has been in the queue too long.") {
			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}
		if resp.ErrorCode == "4.7.1" {
			resp.RetryValidation = true
		}
	case "501":
		if strings.Contains(resp.Description, "Invalid address") {
			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}
	case "503":
		if strings.Contains(resp.Description, "User unknown") {
			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}
	case "550":
		if strings.Contains(resp.Description, "Invalid Recipient") || strings.Contains(resp.Description, "Recipient not found") {
			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}
		if resp.ErrorCode == "5.2.1" || resp.ErrorCode == "5.7.1" || resp.ErrorCode == "5.1.1" || resp.ErrorCode == "5.1.6" || resp.ErrorCode == "5.1.0" {
			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}

	}
	return resp
}

func catchAllTest(validationRequest EmailValidationRequest) (bool, mailserver.SMPTValidation) {
	_, domain, ok := syntax.GetEmailUserAndDomain(validationRequest.Email)
	if !ok {
		log.Printf("Cannot run Catch-All test, invalid email address")
		return false, mailserver.SMPTValidation{}
	}

	catchAllEmail := fmt.Sprintf("%s@%s", validationRequest.CatchAllTestUser, domain)
	isVerified, smtpValidation, err := mailserver.VerifyEmailAddress(
		catchAllEmail,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		validationRequest.Dns,
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
		firstName, lastName := GenerateNames()
		request.FromEmail = fmt.Sprintf("%s.%s@%s", firstName, lastName, request.FromDomain)
	}
	if request.CatchAllTestUser == "" {
		request.CatchAllTestUser = GenerateCatchAllUsername()
	}
	return nil
}
