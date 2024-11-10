package cli

import (
	"fmt"
	"os"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/syntax"
	"github.com/customeros/mailsherpa/internal/util"
	"github.com/customeros/mailsherpa/mailvalidate"
)

type VerifyEmailResponse struct {
	Email                 string
	Deliverable           string
	IsValidSyntax         bool
	IsCatchAll            bool
	Provider              string
	SecureGatewayProvider string
	IsRisky               bool
	Risk                  VerifyEmailRisk
	Syntax                mailvalidate.SyntaxValidation
	AlternateEmail        mailvalidate.AlternateEmail
	RetryValidation       bool
	Smtp                  mailvalidate.SmtpResponse
	MailServerHealth      mailvalidate.MailServerHealth
}

type VerifyEmailRisk struct {
	IsFirewalled    bool
	IsFreeAccount   bool
	IsRoleAccount   bool
	IsMailboxFull   bool
	IsPrimaryDomain bool
}

func BuildRequest(email string) mailvalidate.EmailValidationRequest {
	fromEmail, fromDomain := util.GenerateSenderEmail()

	ok, cleanEmail, _, domain := syntax.NormalizeEmailAddress(email)
	if !ok {
		fmt.Println("Invalid email address")
		os.Exit(1)
	}

	dnsFromEmail := domaincheck.CheckDNS(domain)
	request := mailvalidate.EmailValidationRequest{
		Email:            cleanEmail,
		FromDomain:       fromDomain,
		FromEmail:        fromEmail,
		CatchAllTestUser: util.GenerateCatchAllUsername(),
		Dns:              &dnsFromEmail,
	}
	return request
}

func BuildResponse(
	emailAddress string,
	syntax mailvalidate.SyntaxValidation,
	domain mailvalidate.DomainValidation,
	email mailvalidate.EmailValidation,
) VerifyEmailResponse {
	isRisky := false
	if email.IsFreeAccount ||
		email.IsRoleAccount ||
		email.IsMailboxFull ||
		domain.IsFirewalled ||
		!domain.IsPrimaryDomain {

		isRisky = true
	}

	if domain.IsCatchAll {
		email.IsDeliverable = "unknown"
	}

	if syntax.IsSystemGenerated {
		email.IsDeliverable = "false"
		isRisky = true
	}

	risk := VerifyEmailRisk{
		IsFirewalled:    domain.IsFirewalled,
		IsFreeAccount:   email.IsFreeAccount,
		IsRoleAccount:   email.IsRoleAccount,
		IsMailboxFull:   email.IsMailboxFull,
		IsPrimaryDomain: domain.IsPrimaryDomain,
	}

	cleanEmail := emailAddress
	if syntax.IsValid {
		cleanEmail = fmt.Sprintf("%s@%s", syntax.User, syntax.Domain)
	}

	response := VerifyEmailResponse{
		Email:                 cleanEmail,
		Deliverable:           email.IsDeliverable,
		IsValidSyntax:         syntax.IsValid,
		IsCatchAll:            domain.IsCatchAll,
		Provider:              domain.Provider,
		SecureGatewayProvider: domain.SecureGatewayProvider,
		IsRisky:               isRisky,
		Risk:                  risk,
		AlternateEmail:        email.AlternateEmail,
		RetryValidation:       email.RetryValidation,
		Syntax:                syntax,
		Smtp:                  email.SmtpResponse,
		MailServerHealth:      email.MailServerHealth,
	}

	return response
}
