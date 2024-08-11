package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/customeros/mailsherpa/bulkvalidate"
	"github.com/customeros/mailsherpa/internal/run"
	"github.com/customeros/mailsherpa/mailvalidate"
)

func PrintUsage() {
	fmt.Println("Usage: mailsherpa <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  <email>")
	fmt.Println("  bulk <input file> <output file>")
	fmt.Println("  domain <domain>")
	fmt.Println("  syntax <email>")
	fmt.Println("  version")
}

func BulkVerify(inputFilePath, outputFilePath string) {
	bulkvalidate.RunBulkValidation(inputFilePath, outputFilePath)
}

func VerifyDomain(domain string, printResults bool) mailvalidate.DomainValidation {
	request := run.BuildRequest(fmt.Sprintf("user@%s", domain))
	domainResults, err := mailvalidate.ValidateDomain(request, true)
	if err != nil {
		fmt.Println(err)
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
	emailResults, err := mailvalidate.ValidateEmail(request)
	if err != nil {
		fmt.Println(err)
	}

	response := run.BuildResponse(email, syntaxResults, domainResults, emailResults)
	printOutput(response)
}

func Version() {
	fmt.Println("MailSherpa", run.Version)
}

func printOutput(response interface{}) {
	jsonData, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	fmt.Println(string(jsonData))
}
