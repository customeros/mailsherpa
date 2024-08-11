package run

import (
	"fmt"

	"github.com/customeros/mailsherpa/mailvalidate"
)

const (
	fromDomain            = "gmail.com"
	validateFreeAccounts  = true
	validateRoleMailboxes = true
	Version               = "0.0.10"
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

	request := mailvalidate.EmailValidationRequest{
		Email:                email,
		FromDomain:           fromDomain,
		FromEmail:            fmt.Sprintf("%s.%s@%s", firstname, lastname, fromDomain),
		CatchAllTestUser:     mailvalidate.GenerateCatchAllUsername(),
		ValidateFreeAccounts: validateFreeAccounts,
		ValidateRoleAccounts: validateRoleMailboxes,
	}
	return request
}

func BuildResponse(emailAddress string, syntax mailvalidate.SyntaxValidation, domain mailvalidate.DomainValidation, email mailvalidate.EmailValidation) VerifyEmailResponse {
	isRisky := false
	if email.IsFreeAccount || email.IsRoleAccount || email.IsMailboxFull || domain.IsCatchAll || domain.IsFirewalled {
		isRisky = true
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

	response := VerifyEmailResponse{
		Email:         emailAddress,
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
