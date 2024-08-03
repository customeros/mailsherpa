package validate

import (
	"log"

	"github.com/customeros/mailhawk/internal/mx"
	"github.com/customeros/mailhawk/internal/smtp"
	"github.com/customeros/mailhawk/internal/syntax"
)

type EmailValidationRequest struct {
	email              string
	fromDomain         string
	fromEmail          string
	catchAllTestEmail  string
	proxy              smpt.ProxySetup
	roleEmailsFilePath string
	freeEmailsFilePath string
	knownSpfFilePath   string
}

type EmailValidatation struct {
	email             string
	isDeliverable     bool
	provider          string
	authorizedSenders []string
	risk              emailRisk
	syntax            emailSyntax
	smtp              smpt.SMPTValidation
}

type emailRisk struct {
	isRisky       bool
	isFirewalled  bool
	isRoleAccount bool
	isFreeAccount bool
	isCatchAll    bool
}

type emailSyntax struct {
	isValid bool
	user    string
	domain  string
}

func GetEmailValidation(emailToValidate EmailValidationRequest) EmailValidatation {
	var results EmailValidatation
	results.email = emailToValidate.email
	results.syntax = getEmailSyntax(emailToValidate.email)

	mxRecords, err := mx.GetMXRecordsForEmail(emailToValidate.email)
	if err != nil {
		log.Println(err)
	}

	results.provider = mx.GetEmailServiceProviderFromMX(mxRecords)
	results.authorizedSenders, err = mx.GetEmailProvidersFromSPF(emailToValidate.email)
	if err != nil {
		log.Println(err)
	}

	results.isDeliverable, results.smtp, err = smpt.VerifyEmailAddress(
		emailToValidate.email,
		emailToValidate.fromDomain,
		emailToValidate.fromEmail,
		emailToValidate.proxy,
	)
	if err != nil {
		log.Println(err)
	}

	if results.isDeliverable {
		results.risk.isCatchAll, _, _ = smpt.VerifyEmailAddress(
			emailToValidate.catchAllTestEmail,
			emailToValidate.fromDomain,
			emailToValidate.fromEmail,
			emailToValidate.proxy,
		)
	}

	isFirewall := mx.IsFirewall(results.provider)

	return results
}

func getEmailSyntax(email string) emailSyntax {
	var results emailSyntax
	ok := syntax.IsValidEmailSyntax(email)
	if !ok {
		return results
	}

	user, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return results
	}
	results.isValid = true
	results.user = user
	results.domain = domain
	return results
}
