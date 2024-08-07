package main

import (
	"fmt"
	"github.com/customeros/mailsherpa/mailvalidate"
	"os"
)

func mainTest() {
	//datastudy.RunDataStudy("/Users/mbrown/downloads/test.csv", "/Users/mbrown/desktop/results.csv")
}

func main() {
	email := parseArgs()

	// build validation request
	request := mailvalidate.EmailValidationRequest{
		Email:      email,
		FromDomain: "hubspot.com",
	}

	syntaxResults := mailvalidate.ValidateEmailSyntax(email)
	domainResults, err := mailvalidate.ValidateDomain(request, true)
	if err != nil {
		fmt.Println(err)
	}
	emailResults, err := mailvalidate.ValidateEmail(request)
	if err != nil {
		fmt.Println(err)
	}
	buildResponse(syntaxResults, domainResults, emailResults)
}

func parseArgs() string {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <email>")
		os.Exit(1)
	}
	email := os.Args[1]
	return email
}

func buildResponse(syntax mailvalidate.SyntaxValidation, domain mailvalidate.DomainValidation, email mailvalidate.EmailValidation) {
	isRisky := false
	if email.IsFreeAccount || email.IsRoleAccount || email.IsMailboxFull || domain.IsCatchAll || domain.IsFirewalled {
		isRisky = true
	}

	fmt.Println("isDeliverable:", email.IsDeliverable)
	fmt.Println("isValidSyntax:", syntax.IsValid)
	fmt.Println("provider:", domain.Provider)
	fmt.Println("firewall:", domain.Firewall)
	fmt.Println("isRisky:", isRisky)
	fmt.Println("")
	fmt.Println("isFirewalled:", domain.IsFirewalled)
	fmt.Println("isFreeAccount:", email.IsFreeAccount)
	fmt.Println("isRoleAccount:", email.IsRoleAccount)
	fmt.Println("isMailboxFull:", email.IsMailboxFull)
	fmt.Println("isCatchAll:", domain.IsCatchAll)
	fmt.Println("")
	fmt.Println("Authorized Senders:")
	fmt.Println("Enterprise:", domain.AuthorizedSenders.Enterprise)
	fmt.Println("Hosting:", domain.AuthorizedSenders.Hosting)
	fmt.Println("Security:", domain.AuthorizedSenders.Security)
	fmt.Println("Webmail:", domain.AuthorizedSenders.Webmail)
	fmt.Println("")
	fmt.Println("SMTP Errors")
	fmt.Println(email.SmtpError)
}
