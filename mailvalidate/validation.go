package mailvalidate

import (
	"fmt"
	"log"
	"strings"

	"github.com/pkg/errors"
	"github.com/rdegges/go-ipify"

	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/mailserver"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type EmailValidationRequest struct {
	Email            string
	FromDomain       string
	FromEmail        string
	CatchAllTestUser string
	Dns              *dns.DNS
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
	MailServerHealth  MailServerHealth
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
	IsValid    bool
	User       string
	Domain     string
	CleanEmail string
}

type MailServerHealth struct {
	IsGreylisted  bool
	IsBlacklisted bool
	ServerIP      string
}

func ValidateEmailSyntax(email string) SyntaxValidation {
	isValid, cleanEmail := syntax.IsValidEmailSyntax(email)
	if !isValid {
		return SyntaxValidation{}
	}

	user, domain, ok := syntax.GetEmailUserAndDomain(cleanEmail)
	if !ok {
		return SyntaxValidation{}
	}
	return SyntaxValidation{
		IsValid:    true,
		User:       user,
		Domain:     domain,
		CleanEmail: cleanEmail,
	}
}

func ValidateDomain(validationRequest EmailValidationRequest, validateCatchAll bool) (DomainValidation, error) {
	knownProviders, err := dns.GetKnownProviders()
	if err != nil {
		return DomainValidation{}, errors.Wrap(err, "Error getting known providers")
	}
	return ValidateDomainWithCustomKnownProviders(validationRequest, *knownProviders, validateCatchAll)
}

func ValidateDomainWithCustomKnownProviders(validationRequest EmailValidationRequest, knownProviders dns.KnownProviders, validateCatchAll bool) (DomainValidation, error) {
	var results DomainValidation
	if err := validateRequest(&validationRequest); err != nil {
		return results, errors.Wrap(err, "Invalid request")
	}

	if validationRequest.Dns == nil {
		dnsFromEmail := dns.GetDNS(validationRequest.Email)
		validationRequest.Dns = &dnsFromEmail
	}

	if len(validationRequest.Dns.MX) != 0 {
		results.HasMXRecord = true
		provider, firewall := dns.GetEmailProviderFromMx(*validationRequest.Dns, knownProviders)
		results.Provider = provider
		if firewall != "" {
			results.Firewall = firewall
			results.IsFirewalled = true
		}
	}

	if validationRequest.Dns.SPF != "" {
		results.HasSPFRecord = true
		authorizedSenders := dns.GetAuthorizedSenders(*validationRequest.Dns, &knownProviders)
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

		catchAllResults, mailServerHealth, err := catchAllTest(validationRequest)
		if err != nil {
			log.Panicf("Catch-all test failed: %v", err)
		}

		if catchAllResults.IsDeliverable == true {
			results.IsCatchAll = true
			results.CanConnectSMTP = true
		}

		if catchAllResults.SmtpSuccess == true {
			results.CanConnectSMTP = true
		}

		results.MailServerHealth = mailServerHealth
	}

	return results, nil
}

func ValidateEmail(validationRequest EmailValidationRequest) (EmailValidation, MailServerHealth, error) {
	var results EmailValidation

	if validationRequest.Dns == nil {
		dnsFromEmail := dns.GetDNS(validationRequest.Email)
		validationRequest.Dns = &dnsFromEmail
	}

	if err := validateRequest(&validationRequest); err != nil {
		return results, MailServerHealth{}, errors.Wrap(err, "Invalid request")
	}
	emailSyntaxResult := ValidateEmailSyntax(validationRequest.Email)
	if !emailSyntaxResult.IsValid {
		return results, MailServerHealth{}, fmt.Errorf("Invalid email address")
	}

	freeEmails, err := GetFreeEmailList()
	if err != nil {
		return results, MailServerHealth{}, errors.Wrap(err, "Error getting free email list")
	}

	roleAccounts, err := GetRoleAccounts()
	if err != nil {
		return results, MailServerHealth{}, errors.Wrap(err, "Error getting role accounts")
	}

	email := fmt.Sprintf("%s@%s", emailSyntaxResult.User, emailSyntaxResult.Domain)

	isFreeEmail, err := IsFreeEmailCheck(email, freeEmails)
	if err != nil {
		return results, MailServerHealth{}, errors.Wrap(err, "Error executing free email check")
	}
	results.IsFreeAccount = isFreeEmail

	isRoleAccount, err := IsRoleAccountCheck(email, roleAccounts)
	if err != nil {
		return results, MailServerHealth{}, errors.Wrap(err, "Error executing role account check")
	}
	results.IsRoleAccount = isRoleAccount

	smtpValidation, err := mailserver.VerifyEmailAddress(
		email,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		*validationRequest.Dns,
	)
	if err != nil {
		return results, MailServerHealth{}, errors.Wrap(err, "Error validating email via SMTP")
	}
	results.IsMailboxFull = smtpValidation.InboxFull
	results.ResponseCode = smtpValidation.ResponseCode
	results.ErrorCode = smtpValidation.ErrorCode
	results.Description = smtpValidation.Description
	results.SmtpResponse = smtpValidation.SmtpResponse

	finalResults, serverHealth := handleSmtpResponses(results)

	return finalResults, serverHealth, nil
}

func handleSmtpResponses(resp EmailValidation) (EmailValidation, MailServerHealth) {
	var health MailServerHealth

	switch resp.ResponseCode {
	case "250", "251":
		resp.IsDeliverable = true
		resp.SmtpSuccess = true
		resp.ErrorCode = ""
		resp.Description = ""

	case "450", "451", "452":
		resp.RetryValidation = true

		if strings.Contains(resp.Description, "user is over quota") ||
			strings.Contains(resp.Description, "out of storage") {

			resp.IsMailboxFull = true
			resp.SmtpSuccess = true
			resp.RetryValidation = false
		}

		if strings.Contains(resp.Description, "Account inbounds disabled") ||
			strings.Contains(resp.Description, "Relay access denied") ||
			strings.Contains(resp.Description, "Temporary recipient validation error") ||
			strings.Contains(resp.Description, "unverified address") ||
			resp.ErrorCode == "4.4.4" ||
			resp.ErrorCode == "4.2.2" {

			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
		}

		if strings.Contains(resp.Description, "Account service is temporarily unavailable") ||
			strings.Contains(resp.Description, "Greylisted") ||
			strings.Contains(resp.Description, "Greylisting") ||
			strings.Contains(resp.Description, "Internal resource temporarily unavailable") ||
			strings.Contains(resp.Description, "Internal resources are temporarily unavailable") ||
			strings.Contains(resp.Description, "IP Temporarily Blacklisted") ||
			strings.Contains(resp.Description, "Not allowed") ||
			strings.Contains(resp.Description, "not yet authorized to deliver mail from") ||
			strings.Contains(resp.Description, "please retry later") ||
			strings.Contains(resp.Description, "Please try again later") ||
			strings.Contains(resp.Description, "Recipient Temporarily Unavailable") {

			health.IsGreylisted = true
			ip, err := ipify.GetIp()
			if err != nil {
				log.Println("Unable to obtain Mailserver IP")
			}
			health.ServerIP = ip
			resp.SmtpSuccess = false
			resp.RetryValidation = true
		}

	case "501", "503", "550", "551", "552", "554", "557":
		resp.RetryValidation = true

		if strings.Contains(resp.Description, "Access denied, banned sender") ||
			strings.Contains(resp.Description, "barracudanetworks.com/reputation") ||
			strings.Contains(resp.Description, "black list") ||
			strings.Contains(resp.Description, "Blocked") ||
			strings.Contains(resp.Description, "blocked using") ||
			strings.Contains(resp.Description, "envelope blocked") ||
			strings.Contains(resp.Description, "ERS-DUL") ||
			strings.Contains(resp.Description, "Listed by PBL") ||
			strings.Contains(resp.Description, "rejected by Abusix blacklist") {

			health.IsBlacklisted = true
			ip, err := ipify.GetIp()
			if err != nil {
				log.Println("Unable to obtain Mailserver IP")
			}
			health.ServerIP = ip
			resp.SmtpSuccess = false
			resp.RetryValidation = true
		}

		if strings.Contains(resp.Description, "user is over quota") ||
			strings.Contains(resp.Description, "out of storage") {

			resp.IsMailboxFull = true
			resp.SmtpSuccess = true
			resp.RetryValidation = false
		}

		if strings.Contains(resp.Description, "Address unknown") ||
			strings.Contains(resp.Description, "Bad address syntax") ||
			strings.Contains(resp.Description, "cannot deliver mail") ||
			strings.Contains(resp.Description, "could not deliver mail") ||
			strings.Contains(resp.Description, "dosn't exist") ||
			strings.Contains(resp.Description, "I am no longer") ||
			strings.Contains(resp.Description, "Invalid address") ||
			strings.Contains(resp.Description, "Invalid Recipient") ||
			strings.Contains(resp.Description, "Invalid recipient") ||
			strings.Contains(resp.Description, "Mailbox not found") ||
			strings.Contains(resp.Description, "mailbox unavailable") ||
			strings.Contains(resp.Description, "mail server could not deliver") ||
			strings.Contains(resp.Description, "message was not delivered") ||
			strings.Contains(resp.Description, "no longer being monitored") ||
			strings.Contains(resp.Description, "no mailbox by that name") ||
			strings.Contains(resp.Description, "No such ID") ||
			strings.Contains(resp.Description, "No such local user") ||
			strings.Contains(resp.Description, "No such user") ||
			strings.Contains(resp.Description, "No Such User Here") ||
			strings.Contains(resp.Description, "Recipient not found") ||
			strings.Contains(resp.Description, "Relay not allowed") ||
			strings.Contains(resp.Description, "relay not permitted") ||
			strings.Contains(resp.Description, "Relaying denied") ||
			strings.Contains(resp.Description, "relaying denied") ||
			strings.Contains(resp.Description, "that domain isn't in my list of allowed rcpthosts") ||
			strings.Contains(resp.Description, "Unknown user") ||
			strings.Contains(resp.Description, "unmonitored inbox") ||
			strings.Contains(resp.Description, "Unroutable address") ||
			strings.Contains(resp.Description, "User unknown") ||
			strings.Contains(resp.Description, "User not found") ||
			strings.Contains(resp.Description, "verify address failed") ||
			strings.Contains(resp.Description, "We do not relay") ||
			strings.Contains(resp.Description, "_403") ||
			resp.ErrorCode == "5.0.0" ||
			resp.ErrorCode == "5.0.1" ||
			resp.ErrorCode == "5.1.0" ||
			resp.ErrorCode == "5.1.1" ||
			resp.ErrorCode == "5.1.6" ||
			resp.ErrorCode == "5.2.0" ||
			resp.ErrorCode == "5.2.1" ||
			resp.ErrorCode == "5.4.1" ||
			resp.ErrorCode == "5.5.1" {

			resp.SmtpSuccess = true
			resp.ErrorCode = ""
			resp.Description = ""
			resp.RetryValidation = false
		}

	}
	return resp, health
}

func catchAllTest(validationRequest EmailValidationRequest) (EmailValidation, MailServerHealth, error) {
	var results EmailValidation
	var health MailServerHealth

	_, domain, ok := syntax.GetEmailUserAndDomain(validationRequest.Email)
	if !ok {
		log.Printf("Cannot run Catch-All test, invalid email address")
		return results, health, nil
	}

	catchAllEmail := fmt.Sprintf("%s@%s", validationRequest.CatchAllTestUser, domain)

	smtpValidation, err := mailserver.VerifyEmailAddress(
		catchAllEmail,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		*validationRequest.Dns,
	)
	if err != nil {
		return results, health, err
	}
	results.IsMailboxFull = smtpValidation.InboxFull
	results.ResponseCode = smtpValidation.ResponseCode
	results.ErrorCode = smtpValidation.ErrorCode
	results.Description = smtpValidation.Description
	results.SmtpResponse = smtpValidation.SmtpResponse

	finalResults, serverHealth := handleSmtpResponses(results)

	return finalResults, serverHealth, nil
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
