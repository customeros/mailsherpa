package run

import (
	"fmt"
	"os"

	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/mailvalidate"
)

type VerifyEmailResponse struct {
	Email         string
	IsDeliverable bool
	IsValidSyntax bool
	Provider      string
	Firewall      string
	IsRisky       bool
	Risk          VerifyEmailRisk
	Syntax        mailvalidate.SyntaxValidation
	Smtp          Smtp
}

type VerifyEmailRisk struct {
	IsFirewalled  bool
	IsFreeAccount bool
	IsRoleAccount bool
	IsMailboxFull bool
	IsCatchAll    bool
}

type Smtp struct {
	Success      bool
	Retry        bool
	ResponseCode string
	ErrorCode    string
	Description  string
}

func BuildRequest(email string) mailvalidate.EmailValidationRequest {
	firstname, lastname := mailvalidate.GenerateNames()
	fromDomain, exists := os.LookupEnv("MAIL_SERVER_DOMAIN")
	if !exists {
		fmt.Println("MAIL_SERVER_DOMAIN environment variable not set")
		os.Exit(1)
	}

	request := mailvalidate.EmailValidationRequest{
		Email:            email,
		FromDomain:       fromDomain,
		FromEmail:        fmt.Sprintf("%s.%s@%s", firstname, lastname, fromDomain),
		CatchAllTestUser: mailvalidate.GenerateCatchAllUsername(),
		Dns:              dns.GetDNS(email),
	}
	return request
}

func BuildResponse(emailAddress string, syntax mailvalidate.SyntaxValidation, domain mailvalidate.DomainValidation, email mailvalidate.EmailValidation) VerifyEmailResponse {
	isRisky := false
	if email.IsFreeAccount || email.IsRoleAccount || email.IsMailboxFull || domain.IsCatchAll || domain.IsFirewalled {
		isRisky = true
	}

	if !domain.HasMXRecord {
		email.SmtpSuccess = true
	}

	risk := VerifyEmailRisk{
		IsFirewalled:  domain.IsFirewalled,
		IsFreeAccount: email.IsFreeAccount,
		IsRoleAccount: email.IsRoleAccount,
		IsMailboxFull: email.IsMailboxFull,
		IsCatchAll:    domain.IsCatchAll,
	}

	smtp := Smtp{
		Success:      email.SmtpSuccess,
		Retry:        email.RetryValidation,
		ResponseCode: email.ResponseCode,
		ErrorCode:    email.ErrorCode,
		Description:  email.Description,
	}

	cleanEmail := emailAddress
	if syntax.IsValid {
		cleanEmail = fmt.Sprintf("%s@%s", syntax.User, syntax.Domain)
	}

	response := VerifyEmailResponse{
		Email:         cleanEmail,
		IsDeliverable: email.IsDeliverable,
		Provider:      domain.Provider,
		Firewall:      domain.Firewall,
		IsRisky:       isRisky,
		Risk:          risk,
		Syntax:        syntax,
		Smtp:          smtp,
	}

	return response
}
