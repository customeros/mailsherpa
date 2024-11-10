package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/customeros/mailsherpa/internal/run"
	"github.com/customeros/mailsherpa/mailvalidate"
)

var version = "dev"

func PrintUsage() {
	fmt.Println("Usage: mailsherpa <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  <email>")
	fmt.Println("  domain <domain>")
	fmt.Println("  syntax <email>")
	fmt.Println("  version")
}

func VerifyDomain(domain string, printResults bool) mailvalidate.DomainValidation {
	request := run.BuildRequest(fmt.Sprintf("user@%s", domain))
	domainResults := mailvalidate.ValidateDomain(request)
	if domainResults.Error != "" {
		fmt.Println(domainResults.Error)
	}

	if printResults {
		printOutput(domainResults)
	}
	return domainResults
}

func VerifySyntax(email string, printResults bool) mailvalidate.SyntaxValidation {
	syntaxResults := mailvalidate.ValidateEmailSyntax(email)

	if printResults {
		printOutput(syntaxResults)
	}
	return syntaxResults
}

func VerifyEmail(email string) {
	request := run.BuildRequest(email)
	syntaxResults := VerifySyntax(email, false)
	domainResults := VerifyDomain(syntaxResults.Domain, false)
	emailResults := mailvalidate.ValidateEmail(request)
	if emailResults.Error != "" {
		fmt.Println(emailResults.Error)
	}

	response := run.BuildResponse(email, syntaxResults, domainResults, emailResults)
	printOutput(response)
}

func Version() {
	fmt.Printf("MailSherpa %s\n", version)
}

func printOutput(response interface{}) {
	jsonData, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	fmt.Println(string(jsonData))
}
