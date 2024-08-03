package main

import (
	"fmt"
	"os"

	"github.com/customeros/mailhawk/internal/checks"
	"github.com/customeros/mailhawk/internal/syntax"
)

type EmailValidation struct {
	email                   string
	isDeliverable           bool
	enterpriseEmailProvider string
	emailProviders          map[string]string
	risk                    EmailRisk
	syntax                  EmailSyntax
}

type EmailRisk struct {
	isRisky       bool
	isFirewalled  bool
	isRoleAccount bool
	isFreeAccount bool
	isCatchAll    bool
}

type EmailSyntax struct {
	user   string
	domain string
}

type Config struct {
	freeEmailsFile     string
	roleAccountsFile   string
	emailProvidersFile string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <email>")
		return
	}
	email := os.Args[1]
	freeList := "/Users/mbrown/src/github.com/customeros/mailhawk/role_emails.toml"
	fmt.Println(checks.IsRoleAccountCheck(email, freeList))
}

func getEmailSyntax(email string) (EmailSyntax, error) {
	var results EmailSyntax
	ok := syntax.IsValidEmailSyntax(email)
	if !ok {
		return results, fmt.Errorf("Email address is invalid")
	}

	user, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return results, fmt.Errorf("Email address is invalid")
	}
	results.user = user
	results.domain = domain
	return results, nil
}

func getEmailRisk(email string, configFiles Config) (EmailRisk, error) {
	var results EmailRisk

	roleAccountCheck, err := checks.IsRoleAccountCheck(email, configFiles.roleAccountsFile)
	if err != nil {
		return results, err
	}

	freeAccountCheck, err := checks.IsFreeEmailCheck(email, configFiles.freeEmailsFile)
	if err != nil {
		return results, err
	}
}
