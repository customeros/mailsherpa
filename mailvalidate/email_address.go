package mailvalidate

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rdegges/go-ipify"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/free_emails"
	"github.com/customeros/mailsherpa/internal/mailserver"
	"github.com/customeros/mailsherpa/internal/role_accounts"
	"github.com/customeros/mailsherpa/internal/syntax"
)

// Explicitly handled SMTP response codes
const (
	deliverableEmailCodes = "250, 251"
	temporaryFailureCodes = "421, 450, 451, 452, 453"
	permanentFailureCodes = "500, 501, 503, 525, 541, 542, 550, 551, 552, 554, 557"
)

type AlternateEmail struct {
	Email string
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

// ValidateEmail performs the main email validation
func ValidateEmail(validationRequest EmailValidationRequest) EmailValidation {
	results := initializeValidationResults()

	// Validate request parameters
	if err := validateRequest(&validationRequest); err != nil {
		results.Error = fmt.Sprintf("Invalid request: %v", err)
		return results
	}

	// Ensure DNS records exist
	if err := ensureDNSRecords(&validationRequest); err != nil {
		results.Error = err.Error()
		return results
	}

	// Perform email checks
	if err := performEmailChecks(&validationRequest, &results); err != nil {
		results.Error = err.Error()
		return results
	}

	// Handle alternate email if needed
	if !results.IsFreeAccount {
		handleAlternateEmail(&validationRequest, &results)
	}

	return results
}

func initializeValidationResults() EmailValidation {
	return EmailValidation{
		IsDeliverable: "unknown",
		SmtpResponse:  SmtpResponse{},
	}
}

func ensureDNSRecords(req *EmailValidationRequest) error {
	if req.Dns == nil {
		_, _, _, domain := syntax.NormalizeEmailAddress(req.Email)
		dns := domaincheck.CheckDNS(domain)
		req.Dns = &dns
	}
	return nil
}

func performEmailChecks(req *EmailValidationRequest, results *EmailValidation) error {
	_, _, username, domain := syntax.NormalizeEmailAddress(req.Email)

	// Check if it's a free email
	if isFree, err := freemail.IsFreeEmailCheck(domain); err != nil {
		return fmt.Errorf("Error running free email check: %v", err)
	} else {
		results.IsFreeAccount = isFree
	}

	// Check if it's a role account
	if isRole, err := roleaccounts.IsRoleAccountCheck(username); err != nil {
		return fmt.Errorf("Error running role account check: %v", err)
	} else {
		results.IsRoleAccount = isRole
	}

	// Perform SMTP validation
	smtpValidation := performSMTPValidation(req)
	updateSMTPResults(results, smtpValidation)

	handleSmtpResponses(req, results)

	return nil
}

func performSMTPValidation(req *EmailValidationRequest) mailserver.SMPTValidation {
	return mailserver.VerifyEmailAddress(
		req.Email,
		req.FromDomain,
		req.FromEmail,
		*req.Dns,
	)
}

func updateSMTPResults(results *EmailValidation, smtpValidation mailserver.SMPTValidation) {
	results.IsMailboxFull = smtpValidation.InboxFull
	results.SmtpResponse = SmtpResponse{
		ResponseCode:   smtpValidation.ResponseCode,
		ErrorCode:      smtpValidation.ErrorCode,
		Description:    smtpValidation.Description,
		CanConnectSMTP: smtpValidation.CanConnectSmtp,
	}
}

func handleAlternateEmail(req *EmailValidationRequest, results *EmailValidation) {
	if req.DomainValidationParams != nil {
		if !req.DomainValidationParams.IsPrimaryDomain && req.DomainValidationParams.PrimaryDomain != "" {
			_, _, username, _ := syntax.NormalizeEmailAddress(req.Email)
			results.AlternateEmail.Email = fmt.Sprintf("%s@%s", username, req.DomainValidationParams.PrimaryDomain)
		}
	}
}

// handleSmtpResponses processes SMTP response codes and descriptions
func handleSmtpResponses(req *EmailValidationRequest, resp *EmailValidation) {
	resp.RetryValidation = true

	if isNoMXRecordError(resp.SmtpResponse.Description) {
		resp.IsDeliverable = "false"
		resp.RetryValidation = false
		return
	}

	switch {
	case isDeliverableResponse(resp.SmtpResponse.ResponseCode):
		handleDeliverableResponse(resp)
	case isTemporaryFailure(resp.SmtpResponse.ResponseCode):
		handleTemporaryFailure(req, resp)
	case isPermanentFailure(resp.SmtpResponse.ResponseCode):
		handlePermanentFailure(req, resp)
	}
}

// Response classification functions
func isDeliverableResponse(code string) bool {
	return strings.Contains(deliverableEmailCodes, code)
}

func isTemporaryFailure(code string) bool {
	return strings.Contains(temporaryFailureCodes, code)
}

func isPermanentFailure(code string) bool {
	return strings.Contains(permanentFailureCodes, code)
}

func isNoMXRecordError(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "no mx records") ||
		strings.Contains(desc, "cannot connect to any mx server")
}

// Response handlers
func handleDeliverableResponse(resp *EmailValidation) {
	resp.IsDeliverable = "true"
	resp.RetryValidation = false
}

func handleTemporaryFailure(req *EmailValidationRequest, resp *EmailValidation) {
	switch {
	case isBlacklistError(resp.SmtpResponse.Description):
		blacklisted(req, resp)
	case isGreylistError(resp.SmtpResponse.Description):
		greylisted(req, resp)
	case isMailboxFullError(resp.SmtpResponse.Description):
		handleMailboxFull(resp)
	case isDeliveryFailure(resp.SmtpResponse.Description, resp.SmtpResponse.ErrorCode):
		handleDeliveryFailure(resp)
	case isTLSError(resp.SmtpResponse.Description):
		handleTLSRequirement(resp)
	}
}

func handlePermanentFailure(req *EmailValidationRequest, resp *EmailValidation) {
	switch {
	case isPermanentBlacklistError(resp.SmtpResponse.Description):
		blacklisted(req, resp)
	case isTemporaryBlockError(resp.SmtpResponse.Description):
		greylisted(req, resp)
	case isMailboxFullError(resp.SmtpResponse.Description):
		handleMailboxFull(resp)
	case isTLSError(resp.SmtpResponse.Description):
		handleTLSRequirement(resp)
	case isRetryableError(resp.SmtpResponse.Description):
		handleRetryableError(resp)
	case isInvalidAddressError(resp.SmtpResponse.Description, resp.SmtpResponse.ErrorCode):
		handleInvalidAddress(resp)
	default:
		// For unhandled permanent failures, mark as unknown
		resp.IsDeliverable = "unknown"
		resp.RetryValidation = false
	}
}

// isInvalidAddressError checks if the SMTP response indicates an invalid address
func isInvalidAddressError(description string, errorCode string) bool {
	invalidErrorCodes := []string{
		"5.0.0", "5.0.1", "5.1.0", "5.1.1", "5.1.6",
		"5.2.0", "5.2.1", "5.4.1", "5.4.4", "5.5.1", "5.7.1",
	}

	for _, code := range invalidErrorCodes {
		if errorCode == code {
			return true
		}
	}

	lowerDesc := strings.ToLower(description)

	invalidAddressKeywords := []string{
		"address does not exist", "address error", "address not",
		"address unknown", "bad address syntax", "can't verify",
		"cannot deliver mail", "could not deliver mail",
		"disabled recipient", "dosn't exist", "does not exist",
		"invalid address", "invalid recipient",
		"mailbox is frozen", "mailbox not found", "mailbox unavailable",
		"no longer being monitored", "no mail box", "no mailbox",
		"no such", "not allowed", "not a known user", "not exist",
		"not found", "not valid", "recipient not found",
		"recipient unknown", "refused", "rejected", "relay access",
		"relay not", "service not available", "unable to find",
		"unknown recipient", "unknown user", "unmonitored inbox",
		"unroutable address", "user doesn't", "user invalid",
		"user not", "user unknown", "verification problem",
		"verify address failed", "we do not relay",
	}

	for _, keyword := range invalidAddressKeywords {
		if strings.Contains(lowerDesc, keyword) {
			return true
		}
	}

	return false
}

// isPermanentBlacklistError checks for permanent blacklist errors
func isPermanentBlacklistError(description string) bool {
	// Convert description to lowercase once for more efficient comparison
	lowerDesc := strings.ToLower(description)

	blacklistKeywords := []string{
		"access denied",
		"bad reputation",
		"barracudanetworks.com/reputation",
		"black list",
		"blacklist",
		"blocked",
		"blocked by rbl",
		"client host blocked",
		"envelope blocked",
		"ers-dul",
		"listed by pbl",
		"rejected by abusix",
		"sender rejected",
		"spf check failed",
		"transaction failed",
		"spamhaus",
		"rbl",
		"pbl",
	}

	for _, keyword := range blacklistKeywords {
		if strings.Contains(lowerDesc, keyword) {
			return true
		}
	}

	return false
}

// isTemporaryBlockError checks if the response indicates a temporary block
func isTemporaryBlockError(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "temporarily blocked")
}

// isRetryableError checks if the error is retryable
func isRetryableError(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "try again")
}

// handleInvalidAddress processes invalid address responses
func handleInvalidAddress(resp *EmailValidation) {
	resp.IsDeliverable = "false"
	resp.RetryValidation = false
}

// handleRetryableError processes retryable errors
func handleRetryableError(resp *EmailValidation) {
	resp.RetryValidation = true
	resp.IsDeliverable = "unknown"
}

// Error classification helpers
func isMailboxFullError(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "insufficient system storage") ||
		strings.Contains(desc, "out of storage") ||
		strings.Contains(desc, "user is over quota")
}

func isDeliveryFailure(description string, errorCode string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "account inbounds disabled") ||
		strings.Contains(desc, "address rejected") ||
		errorCode == "4.4.4" ||
		errorCode == "4.2.2"
}

// isGreylistError checks if the response indicates greylisting
func isGreylistError(description string) bool {
	greylistKeywords := []string{
		"greylisted",
		"greylisting",
		"please retry later",
		"try again later",
		"temporarily deferred",
		"postgrey",
		"try again in",
		"deferred for",
		"internal resource temporarily unavailable",
	}

	for _, keyword := range greylistKeywords {
		if strings.Contains(strings.ToLower(description), keyword) {
			return true
		}
	}
	return false
}

func isTLSError(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "tls")
}

// isBlacklistError checks for blacklist-related errors
func isBlacklistError(description string) bool {
	blacklistKeywords := []string{
		"not in whitelist",
		"sender address rejected",
		"blocked by rbl",
		"listed by pbl",
		"spamhaus",
		"blacklist",
		"blocklist",
		"reputation",
		"blocked for spam",
		"blocked by",
		"client host blocked",
		"ip blocked",
	}

	for _, keyword := range blacklistKeywords {
		if strings.Contains(strings.ToLower(description), keyword) {
			return true
		}
	}
	return false
}

// Handler implementations
func handleMailboxFull(resp *EmailValidation) {
	resp.IsDeliverable = "false"
	resp.IsMailboxFull = true
	resp.RetryValidation = false
}

func handleDeliveryFailure(resp *EmailValidation) {
	resp.IsDeliverable = "false"
	resp.RetryValidation = false
}

func handleTLSRequirement(resp *EmailValidation) {
	resp.SmtpResponse.TLSRequired = true
	resp.RetryValidation = true
}

func blacklisted(req *EmailValidationRequest, resp *EmailValidation) {
	resp.MailServerHealth.IsBlacklisted = true
	if ip, err := ipify.GetIp(); err != nil {
		log.Printf("Unable to obtain Mailserver IP: %v", err)
	} else {
		resp.MailServerHealth.ServerIP = ip
	}
	resp.MailServerHealth.FromEmail = req.FromEmail
}

func greylisted(req *EmailValidationRequest, resp *EmailValidation) {
	minutes := determineGreylistDelay(resp.SmtpResponse.Description)

	resp.MailServerHealth.IsGreylisted = true
	resp.IsDeliverable = "unknown"

	if ip, err := ipify.GetIp(); err != nil {
		log.Printf("Unable to obtain Mailserver IP: %v", err)
	} else {
		resp.MailServerHealth.ServerIP = ip
	}

	resp.MailServerHealth.FromEmail = req.FromEmail
	resp.MailServerHealth.RetryAfter = getRetryTimestamp(minutes)
}

func determineGreylistDelay(description string) int {
	switch {
	case strings.Contains(description, "4 minutes"),
		strings.Contains(description, "5 minutes"),
		strings.Contains(description, "five minutes"):
		return 6
	case strings.Contains(description, "360 seconds"):
		return 7
	case strings.Contains(description, "60 seconds"),
		strings.Contains(description, "1 minute"):
		return 2
	default:
		return 75 // default delay
	}
}

func getRetryTimestamp(minutesDelay int) int {
	return int(time.Now().Add(time.Duration(minutesDelay) * time.Minute).Unix())
}

// CatchAllTest performs catch-all testing for domains
func catchAllTest(validationRequest *EmailValidationRequest) EmailValidation {
	results := initializeValidationResults()

	_, _, _, domain := syntax.NormalizeEmailAddress(validationRequest.Email)
	catchAllEmail := fmt.Sprintf("%s@%s", validationRequest.CatchAllTestUser, domain)

	smtpValidation := performSMTPValidation(&EmailValidationRequest{
		Email:      catchAllEmail,
		FromDomain: validationRequest.FromDomain,
		FromEmail:  validationRequest.FromEmail,
		Dns:        validationRequest.Dns,
	})

	updateSMTPResults(&results, smtpValidation)
	handleSmtpResponses(validationRequest, &results)

	return results
}
