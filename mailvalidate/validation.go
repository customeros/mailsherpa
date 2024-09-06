package mailvalidate

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rdegges/go-ipify"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/mailserver"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type EmailValidationRequest struct {
	Email            string
	FromDomain       string
	FromEmail        string
	CatchAllTestUser string
	Dns              *domaincheck.DNS
	// applicable only for email validation. Pass results from domain validation
	DomainValidationParams *DomainValidationParams
}

type DomainValidationParams struct {
	IsPrimaryDomain bool
	PrimaryDomain   string
}

type SyntaxValidation struct {
	IsValid    bool
	User       string
	Domain     string
	CleanEmail string
}

type AlternateEmail struct {
	Email string
}

type DomainValidation struct {
	Provider              string
	SecureGatewayProvider string
	AuthorizedSenders     dns.AuthorizedSenders
	IsFirewalled          bool
	IsCatchAll            bool
	IsPrimaryDomain       bool
	PrimaryDomain         string
	HasMXRecord           bool
	HasSPFRecord          bool
	SmtpResponse          SmtpResponse
	MailServerHealth      MailServerHealth
	Error                 string
}

type EmailValidation struct {
	IsDeliverable    string
	IsMailboxFull    bool
	IsRoleAccount    bool
	IsFreeAccount    bool
	RetryValidation  bool
	SmtpResponse     SmtpResponse
	MailServerHealth MailServerHealth
	AlternateEmail   AlternateEmail
	Error            string
}

type MailServerHealth struct {
	IsGreylisted  bool
	IsBlacklisted bool
	ServerIP      string
	FromEmail     string
	RetryAfter    int
}

type SmtpResponse struct {
	CanConnectSMTP bool
	TLSRequired    bool
	ResponseCode   string
	ErrorCode      string
	Description    string
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

func ValidateDomain(validationRequest EmailValidationRequest) DomainValidation {
	var results DomainValidation
	knownProviders, err := dns.GetKnownProviders()
	if err != nil {
		results.Error = fmt.Sprintf("Error getting known providers: %v", err)
		return results
	}
	return ValidateDomainWithCustomKnownProviders(validationRequest, *knownProviders)
}

func ValidateDomainWithCustomKnownProviders(validationRequest EmailValidationRequest, knownProviders dns.KnownProviders) DomainValidation {
	var results DomainValidation
	if err := validateRequest(&validationRequest); err != nil {
		results.Error = fmt.Sprintf("Invalid request: %v", err)
		return results
	}

	if validationRequest.Dns == nil {
		dnsFromEmail := dns.GetDNS(validationRequest.Email)
		validationRequest.Dns = &dnsFromEmail
	}

	evaluateDnsRecords(&validationRequest, &knownProviders, &results)

	_, domain, ok := syntax.GetEmailUserAndDomain(validationRequest.Email)
	if !ok {
		results.Error = fmt.Sprintf("Invalid Email Address")
		return results
	}

	redirects, primaryDomain := domaincheck.CheckRedirects(domain)
	if !redirects && validationRequest.Dns.CNAME == "" && results.HasMXRecord && validationRequest.Dns.HasA {
		results.IsPrimaryDomain = true
	} else {
		results.PrimaryDomain = primaryDomain
	}

	catchAllResults := catchAllTest(&validationRequest)

	if catchAllResults.IsDeliverable == "true" {
		results.IsCatchAll = true
	}

	results.MailServerHealth = catchAllResults.MailServerHealth
	results.SmtpResponse = catchAllResults.SmtpResponse

	return results
}

func ValidateEmail(validationRequest EmailValidationRequest) EmailValidation {
	var results EmailValidation
	results.IsDeliverable = "unknown"

	if validationRequest.Dns == nil {
		dnsFromEmail := dns.GetDNS(validationRequest.Email)
		validationRequest.Dns = &dnsFromEmail
	}

	if err := validateRequest(&validationRequest); err != nil {
		results.Error = fmt.Sprintf("Invalid request: %v", err)
		return results
	}

	emailSyntaxResult := ValidateEmailSyntax(validationRequest.Email)
	if !emailSyntaxResult.IsValid {
		results.Error = "Invalid email address"
		results.IsDeliverable = "false"
		return results
	}

	freeEmails, err := GetFreeEmailList()
	if err != nil {
		results.Error = fmt.Sprintf("Error getting free email list: %v", err)
		return results
	}

	roleAccounts, err := GetRoleAccounts()
	if err != nil {
		results.Error = fmt.Sprintf("Error getting role accounts list: %v", err)
		return results
	}

	email := emailSyntaxResult.CleanEmail

	isFreeEmail, err := IsFreeEmailCheck(email, &freeEmails)
	if err != nil {
		results.Error = fmt.Sprintf("Error running free email check: %v", err)
		return results
	}
	results.IsFreeAccount = isFreeEmail

	isRoleAccount, err := IsRoleAccountCheck(email, &roleAccounts)
	if err != nil {
		results.Error = fmt.Sprintf("Error running role account check: %v", err)
		return results
	}
	results.IsRoleAccount = isRoleAccount

	smtpValidation := mailserver.VerifyEmailAddress(
		email,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		*validationRequest.Dns,
	)

	results.IsMailboxFull = smtpValidation.InboxFull

	results.SmtpResponse.ResponseCode = smtpValidation.ResponseCode
	results.SmtpResponse.ErrorCode = smtpValidation.ErrorCode
	results.SmtpResponse.Description = smtpValidation.Description
	results.SmtpResponse.CanConnectSMTP = smtpValidation.CanConnectSmtp

	handleSmtpResponses(&validationRequest, &results)

	if validationRequest.DomainValidationParams != nil {
		if !validationRequest.DomainValidationParams.IsPrimaryDomain && validationRequest.DomainValidationParams.PrimaryDomain != "" {
			results.AlternateEmail.Email = fmt.Sprintf("%s@%s", emailSyntaxResult.User, validationRequest.DomainValidationParams.PrimaryDomain)
		}
	}

	return results
}

func evaluateDnsRecords(validationRequest *EmailValidationRequest, knownProviders *dns.KnownProviders, results *DomainValidation) {
	if len(validationRequest.Dns.MX) != 0 {
		results.HasMXRecord = true
		provider, firewall := dns.GetEmailProviderFromMx(*validationRequest.Dns, *knownProviders)
		results.Provider = provider
		if firewall != "" {
			results.SecureGatewayProvider = firewall
			results.IsFirewalled = true
		}
	}

	if validationRequest.Dns.SPF != "" {
		results.HasSPFRecord = true
		authorizedSenders := dns.GetAuthorizedSenders(*validationRequest.Dns, knownProviders)
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
		results.SecureGatewayProvider = results.AuthorizedSenders.Security[0]
	}
}

func handleSmtpResponses(req *EmailValidationRequest, resp *EmailValidation) {
	resp.RetryValidation = true

	if strings.Contains(resp.SmtpResponse.Description, "No MX records") ||
		strings.Contains(resp.SmtpResponse.Description, "Cannot connect to any MX server") {

		resp.IsDeliverable = "false"
		resp.RetryValidation = false
	}

	switch resp.SmtpResponse.ResponseCode {
	case "250", "251":
		resp.IsDeliverable = "true"
		resp.RetryValidation = false

	case "421", "450", "451", "452", "453":

		if strings.Contains(resp.SmtpResponse.Description, "Insufficient system storage") ||
			strings.Contains(resp.SmtpResponse.Description, "out of storage") ||
			strings.Contains(resp.SmtpResponse.Description, "user is over quota") {

			resp.IsDeliverable = "false"
			resp.IsMailboxFull = true
			resp.RetryValidation = false
		}

		if strings.Contains(resp.SmtpResponse.Description, "Account inbounds disabled") ||
			strings.Contains(resp.SmtpResponse.Description, "address rejected") ||
			strings.Contains(resp.SmtpResponse.Description, "Not verified") ||
			strings.Contains(resp.SmtpResponse.Description, "Open relay not allowed") ||
			strings.Contains(resp.SmtpResponse.Description, "Recipient address rejected") ||
			strings.Contains(resp.SmtpResponse.Description, "Recipient not in route") ||
			strings.Contains(resp.SmtpResponse.Description, "Relay access denied") ||
			strings.Contains(resp.SmtpResponse.Description, "relay not") ||
			strings.Contains(resp.SmtpResponse.Description, "Syntax error") ||
			strings.Contains(resp.SmtpResponse.Description, "Temporary recipient validation error") ||
			strings.Contains(resp.SmtpResponse.Description, "unverified address") ||
			strings.Contains(resp.SmtpResponse.Description, "unable to verify user") ||
			strings.Contains(resp.SmtpResponse.Description, "you are trying to use me") ||
			resp.SmtpResponse.ErrorCode == "4.4.4" ||
			resp.SmtpResponse.ErrorCode == "4.2.2" {

			resp.IsDeliverable = "false"
			resp.RetryValidation = false
		}

		if strings.Contains(resp.SmtpResponse.Description, "Account service is temporarily unavailable") ||
			strings.Contains(resp.SmtpResponse.Description, "greylisted") ||
			strings.Contains(resp.SmtpResponse.Description, "Greylisted") ||
			strings.Contains(resp.SmtpResponse.Description, "Greylisting") ||
			strings.Contains(resp.SmtpResponse.Description, "Internal resource temporarily unavailable") ||
			strings.Contains(resp.SmtpResponse.Description, "Internal resources are temporarily unavailable") ||
			strings.Contains(resp.SmtpResponse.Description, "ip and spf record not match") ||
			strings.Contains(resp.SmtpResponse.Description, "IP Temporarily Blacklisted") ||
			strings.Contains(resp.SmtpResponse.Description, "Not allowed") ||
			strings.Contains(resp.SmtpResponse.Description, "not yet authorized to deliver mail from") ||
			strings.Contains(resp.SmtpResponse.Description, "please retry later") ||
			strings.Contains(resp.SmtpResponse.Description, "Please try again later") ||
			strings.Contains(resp.SmtpResponse.Description, "Recipient Temporarily Unavailable") ||
			strings.Contains(resp.SmtpResponse.Description, "temporarily delayed") ||
			strings.Contains(resp.SmtpResponse.Description, "too many") ||
			strings.Contains(resp.SmtpResponse.Description, "domain is not configured") {

			greylisted(req, resp)
		}

		if strings.Contains(resp.SmtpResponse.Description, "TLS") {
			resp.SmtpResponse.TLSRequired = true
			resp.RetryValidation = true
		}

		if strings.Contains(resp.SmtpResponse.Description, "not in whitelist") ||
			strings.Contains(resp.SmtpResponse.Description, "Sender address rejected") {
			blacklisted(req, resp)
		}

	case "500", "501", "503", "525", "541", "542", "550", "551", "552", "554", "557":

		if strings.Contains(resp.SmtpResponse.Description, "exceeded storage") ||
			strings.Contains(resp.SmtpResponse.Description, "Mailbox is full") ||
			strings.Contains(resp.SmtpResponse.Description, "out of storage") ||
			strings.Contains(resp.SmtpResponse.Description, "user is over quota") {

			resp.IsDeliverable = "false"
			resp.IsMailboxFull = true
			resp.RetryValidation = false
		}

		if strings.Contains(resp.SmtpResponse.Description, "Address does not exist") ||
			strings.Contains(resp.SmtpResponse.Description, "Address Error") ||
			strings.Contains(resp.SmtpResponse.Description, "Address not") ||
			strings.Contains(resp.SmtpResponse.Description, "Address unknown") ||
			strings.Contains(resp.SmtpResponse.Description, "Bad address syntax") ||
			strings.Contains(resp.SmtpResponse.Description, "can't verify") ||
			strings.Contains(resp.SmtpResponse.Description, "cannot deliver mail") ||
			strings.Contains(resp.SmtpResponse.Description, "could not deliver mail") ||
			strings.Contains(resp.SmtpResponse.Description, "Disabled recipient") ||
			strings.Contains(resp.SmtpResponse.Description, "dosn't exist") ||
			strings.Contains(resp.SmtpResponse.Description, "DOES NOT EXIST") ||
			strings.Contains(resp.SmtpResponse.Description, "I am no longer") ||
			strings.Contains(resp.SmtpResponse.Description, "I am no longer") ||
			strings.Contains(resp.SmtpResponse.Description, "Invalid address") ||
			strings.Contains(resp.SmtpResponse.Description, "invalid address") ||
			strings.Contains(resp.SmtpResponse.Description, "Invalid Recipient") ||
			strings.Contains(resp.SmtpResponse.Description, "Invalid recipient") ||
			strings.Contains(resp.SmtpResponse.Description, "Mailbox does not exist") ||
			strings.Contains(resp.SmtpResponse.Description, "Mailbox is frozen") ||
			strings.Contains(resp.SmtpResponse.Description, "Mailbox not found") ||
			strings.Contains(resp.SmtpResponse.Description, "mailbox unavailable") ||
			strings.Contains(resp.SmtpResponse.Description, "mailbox is unavailable") ||
			strings.Contains(resp.SmtpResponse.Description, "mail server could not deliver") ||
			strings.Contains(resp.SmtpResponse.Description, "message was not delivered") ||
			strings.Contains(resp.SmtpResponse.Description, "no longer being monitored") ||
			strings.Contains(resp.SmtpResponse.Description, "no mail box") ||
			strings.Contains(resp.SmtpResponse.Description, "no mailbox") ||
			strings.Contains(resp.SmtpResponse.Description, "no such") ||
			strings.Contains(resp.SmtpResponse.Description, "No such") ||
			strings.Contains(resp.SmtpResponse.Description, "No Such") ||
			strings.Contains(resp.SmtpResponse.Description, "Not allowed") ||
			strings.Contains(resp.SmtpResponse.Description, "not a known user") ||
			strings.Contains(resp.SmtpResponse.Description, "not exist") ||
			strings.Contains(resp.SmtpResponse.Description, "not found") ||
			strings.Contains(resp.SmtpResponse.Description, "not valid") ||
			strings.Contains(resp.SmtpResponse.Description, "Recipient not found") ||
			strings.Contains(resp.SmtpResponse.Description, "Recipient unknown") ||
			strings.Contains(resp.SmtpResponse.Description, "refused") ||
			strings.Contains(resp.SmtpResponse.Description, "rejected") ||
			strings.Contains(resp.SmtpResponse.Description, "Rejected") ||
			strings.Contains(resp.SmtpResponse.Description, "REJECTED") ||
			strings.Contains(resp.SmtpResponse.Description, "Relay access") ||
			strings.Contains(resp.SmtpResponse.Description, "Relay not") ||
			strings.Contains(resp.SmtpResponse.Description, "relay not") ||
			strings.Contains(resp.SmtpResponse.Description, "Relaying") ||
			strings.Contains(resp.SmtpResponse.Description, "relaying") ||
			strings.Contains(resp.SmtpResponse.Description, "Service not available") ||
			strings.Contains(resp.SmtpResponse.Description, "that domain isn't in my list of allowed rcpthosts") ||
			strings.Contains(resp.SmtpResponse.Description, "unable to find") ||
			strings.Contains(resp.SmtpResponse.Description, "Unknown recipient") ||
			strings.Contains(resp.SmtpResponse.Description, "Unknown user") ||
			strings.Contains(resp.SmtpResponse.Description, "unknown") ||
			strings.Contains(resp.SmtpResponse.Description, "unmonitored inbox") ||
			strings.Contains(resp.SmtpResponse.Description, "Unroutable address") ||
			strings.Contains(resp.SmtpResponse.Description, "user doesn't") ||
			strings.Contains(resp.SmtpResponse.Description, "user invalid") ||
			strings.Contains(resp.SmtpResponse.Description, "User not") ||
			strings.Contains(resp.SmtpResponse.Description, "User unknown") ||
			strings.Contains(resp.SmtpResponse.Description, "User Unknown") ||
			strings.Contains(resp.SmtpResponse.Description, "verification problem") ||
			strings.Contains(resp.SmtpResponse.Description, "verify address failed") ||
			strings.Contains(resp.SmtpResponse.Description, "We do not relay") ||
			strings.Contains(resp.SmtpResponse.Description, "_403") ||
			resp.SmtpResponse.ErrorCode == "5.0.0" ||
			resp.SmtpResponse.ErrorCode == "5.0.1" ||
			resp.SmtpResponse.ErrorCode == "5.1.0" ||
			resp.SmtpResponse.ErrorCode == "5.1.1" ||
			resp.SmtpResponse.ErrorCode == "5.1.6" ||
			resp.SmtpResponse.ErrorCode == "5.2.0" ||
			resp.SmtpResponse.ErrorCode == "5.2.1" ||
			resp.SmtpResponse.ErrorCode == "5.4.1" ||
			resp.SmtpResponse.ErrorCode == "5.4.4" ||
			resp.SmtpResponse.ErrorCode == "5.5.1" ||
			resp.SmtpResponse.ErrorCode == "5.7.1" {

			resp.IsDeliverable = "false"
			resp.RetryValidation = false
		}

		if strings.Contains(resp.SmtpResponse.Description, "Access denied, banned sender") ||
			strings.Contains(resp.SmtpResponse.Description, "Access Denied") ||
			strings.Contains(resp.SmtpResponse.Description, "bad reputation") ||
			strings.Contains(resp.SmtpResponse.Description, "barracudanetworks.com/reputation") ||
			strings.Contains(resp.SmtpResponse.Description, "black list") ||
			strings.Contains(resp.SmtpResponse.Description, "Blocked") ||
			strings.Contains(resp.SmtpResponse.Description, "blocked") ||
			strings.Contains(resp.SmtpResponse.Description, "envelope blocked") ||
			strings.Contains(resp.SmtpResponse.Description, "ERS-DUL") ||
			strings.Contains(resp.SmtpResponse.Description, "Listed by PBL") ||
			strings.Contains(resp.SmtpResponse.Description, "rejected by Abusix blacklist") ||
			strings.Contains(resp.SmtpResponse.Description, "sender rejected") ||
			strings.Contains(resp.SmtpResponse.Description, "spf check failed") ||
			strings.Contains(resp.SmtpResponse.Description, "Transaction failed") {

			blacklisted(req, resp)
		}

		if strings.Contains(resp.SmtpResponse.Description, "temporarily blocked") {
			greylisted(req, resp)
		}

		if strings.Contains(resp.SmtpResponse.Description, "authentication") ||
			strings.Contains(resp.SmtpResponse.Description, "Authentication") ||
			strings.Contains(resp.SmtpResponse.Description, "encryption") ||
			strings.Contains(resp.SmtpResponse.Description, "TLS") ||
			strings.Contains(resp.SmtpResponse.Description, "tls") ||
			strings.Contains(resp.SmtpResponse.Description, "secured Connection") {
			resp.SmtpResponse.TLSRequired = true
			resp.RetryValidation = true
		}

		if strings.Contains(resp.SmtpResponse.Description, "try again") {
			resp.RetryValidation = true
			resp.IsDeliverable = "unknown"
		}
	}
}

func greylisted(req *EmailValidationRequest, resp *EmailValidation) {

	greylistMinutesBeforeRetry := 75

	resp.MailServerHealth.IsGreylisted = true
	resp.IsDeliverable = "unknown"
	ip, err := ipify.GetIp()
	if err != nil {
		log.Println("Unable to obtain Mailserver IP")
	}

	resp.MailServerHealth.ServerIP = ip
	resp.MailServerHealth.FromEmail = req.FromEmail

	if strings.Contains(resp.SmtpResponse.Description, "4 minutes") ||
		strings.Contains(resp.SmtpResponse.Description, "5 minutes") ||
		strings.Contains(resp.SmtpResponse.Description, "five minutes") {
		greylistMinutesBeforeRetry = 6
	}
	if strings.Contains(resp.SmtpResponse.Description, "360 seconds") {
		greylistMinutesBeforeRetry = 7
	}
	if strings.Contains(resp.SmtpResponse.Description, "60 seconds") ||
		strings.Contains(resp.SmtpResponse.Description, "1 minute") {
		greylistMinutesBeforeRetry = 2
	}
	resp.MailServerHealth.RetryAfter = getRetryTimestamp(greylistMinutesBeforeRetry)
}

func blacklisted(req *EmailValidationRequest, resp *EmailValidation) {

	resp.MailServerHealth.IsBlacklisted = true
	ip, err := ipify.GetIp()
	if err != nil {
		log.Println("Unable to obtain Mailserver IP")
	}
	resp.MailServerHealth.ServerIP = ip
	resp.MailServerHealth.FromEmail = req.FromEmail
}

func catchAllTest(validationRequest *EmailValidationRequest) EmailValidation {
	var results EmailValidation
	results.IsDeliverable = "unknown"

	_, domain, ok := syntax.GetEmailUserAndDomain(validationRequest.Email)
	if !ok {
		results.Error = "Cannot run catch-all test: Invalid email address"
		return results
	}

	catchAllEmail := fmt.Sprintf("%s@%s", validationRequest.CatchAllTestUser, domain)

	smtpValidation := mailserver.VerifyEmailAddress(
		catchAllEmail,
		validationRequest.FromDomain,
		validationRequest.FromEmail,
		*validationRequest.Dns,
	)
	results.IsMailboxFull = smtpValidation.InboxFull
	results.SmtpResponse.ResponseCode = smtpValidation.ResponseCode
	results.SmtpResponse.ErrorCode = smtpValidation.ErrorCode
	results.SmtpResponse.Description = smtpValidation.Description
	results.SmtpResponse.CanConnectSMTP = smtpValidation.CanConnectSmtp

	handleSmtpResponses(validationRequest, &results)

	return results
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

func getRetryTimestamp(minutesDelay int) int {
	currentEpochTime := time.Now().Unix()
	retryTimestamp := time.Unix(currentEpochTime, 0).Add(time.Duration(minutesDelay) * time.Minute).Unix()
	return int(retryTimestamp)
}
