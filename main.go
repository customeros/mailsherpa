package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/customeros/mailsherpa/bulkvalidate"
	"github.com/customeros/mailsherpa/internal/syntax"
	"github.com/customeros/mailsherpa/mailvalidate"
)

const (
	fromDomain            = "gmail.com"
	validateFreeAccounts  = true
	validateRoleMailboxes = true
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		printUsage()
		return
	}

	if args[0] != "verify" {
		fmt.Println("Unknown command.")
		printUsage()
		return
	}

	switch args[1] {
	case "bulk":
		if len(args) != 4 {
			fmt.Println("Usage: mailsherpa verify bulk <input file> <output file>")
			return
		}
		bulkVerify(args[2], args[3])
	case "domain":
		if len(args) != 3 {
			fmt.Println("Usage: mailsherpa verify domain <domain>")
			return
		}
		verifyDomain(args[2])
	case "syntax":
		if len(args) != 3 {
			fmt.Println("Usage: mailsherpa verify syntax <email>")
			return
		}
		verifySyntax(args[2])
	default:
		if len(args) != 2 {
			fmt.Println("Usage: mailsherpa verify <email>")
			return
		}
		verifyEmail(args[1])
	}
}

func printUsage() {
	fmt.Println("Usage: mailhawk <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  verify bulk <input file> <output file>")
	fmt.Println("  verify domain <domain>")
	fmt.Println("  verify syntax <email>")
	fmt.Println("  verify <email>")
}

func bulkVerify(inputFilePath, outputFilePath string) {
	bulkvalidate.RunBulkValidation(inputFilePath, outputFilePath)
}

func verifyDomain(domain string) {
	request := buildRequest(fmt.Sprintf("user@%s", domain))
	domainResults, err := mailvalidate.ValidateDomain(request, true)
	if err != nil {
		fmt.Println("Invalid domain")
	}
	jsonData, err := json.MarshalIndent(domainResults, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	fmt.Println(string(jsonData))
}

func verifySyntax(email string) {
	syntaxResults := mailvalidate.ValidateEmailSyntax(email)
	jsonData, err := json.MarshalIndent(syntaxResults, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	fmt.Println(string(jsonData))
}

func verifyEmail(email string) {
	request := buildRequest(email)
	emailResults, err := mailvalidate.ValidateEmail(request)
	if err != nil {
		fmt.Println(err)
	}
	jsonData, err := json.MarshalIndent(emailResults, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	fmt.Println(string(jsonData))
}

func buildRequest(email string) mailvalidate.EmailValidationRequest {
	firstname, lastname := mailvalidate.GenerateNames()
	_, recipientDomain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		panic("Invalid email")
	}

	request := mailvalidate.EmailValidationRequest{
		Email:                email,
		FromDomain:           fromDomain,
		FromEmail:            fmt.Sprintf("%s.%s@%s", firstname, lastname, fromDomain),
		CatchAllTestUser:     fmt.Sprintf("%s@%S", mailvalidate.GenerateCatchAllUsername(), recipientDomain),
		ValidateFreeAccounts: validateFreeAccounts,
		ValidateRoleAccounts: validateRoleMailboxes,
	}
	return request
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
	fmt.Println("SMTP Response")
	fmt.Println("Success:", email.SmtpSuccess)
	fmt.Println("Retry:", email.RetryValidation)
	fmt.Println(email.ResponseCode)
	fmt.Println(email.ErrorCode)
	fmt.Println(email.Description)
	fmt.Println(email.SmtpResponse)
}
