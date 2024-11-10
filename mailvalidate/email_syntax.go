package mailvalidate

import (
	"fmt"

	"github.com/customeros/mailsherpa/internal/free_emails"
	"github.com/customeros/mailsherpa/internal/role_accounts"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type SyntaxValidation struct {
	Error             string
	IsValid           bool
	User              string
	Domain            string
	CleanEmail        string
	IsRoleAccount     bool
	IsFreeAccount     bool
	IsSystemGenerated bool
}

func ValidateEmailSyntax(email string) SyntaxValidation {
	// Initial syntax validation
	isValid, cleanEmail, user, domain := syntax.NormalizeEmailAddress(email)
	if !isValid {
		return SyntaxValidation{}
	}

	// Create validation result with basic checks
	validation := SyntaxValidation{
		IsValid:           true,
		User:              user,
		Domain:            domain,
		CleanEmail:        cleanEmail,
		IsSystemGenerated: syntax.IsSystemGeneratedUser(user),
	}

	// Check if it's a free email provider
	if isFreeEmail, err := freemail.IsFreeEmailCheck(domain); err != nil {
		validation.Error = fmt.Sprintf("Error running free email check: %s", err.Error())
		return validation
	} else {
		validation.IsFreeAccount = isFreeEmail
	}

	// Check if it's a role account
	if isRoleAccount, err := roleaccounts.IsRoleAccountCheck(user); err != nil {
		validation.Error = fmt.Sprintf("Error running role account check: %s", err.Error())
		return validation
	} else {
		validation.IsRoleAccount = isRoleAccount
	}

	return validation
}
