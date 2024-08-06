package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucasepe/codename"

	"github.com/customeros/mailsherpa/datastudy"
	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/validate"
)

func main() {
	datastudy.RunDataStudy("/Users/mbrown/downloads/bettercontact_test.csv", "/Users/mbrown/desktop/bettercontact_results.csv")
}

func main_old() {
	knownProviders, err := dns.GetKnownProviders("./known_email_providers.toml")
	if err != nil {
		log.Fatal(err)
	}

	freeEmails, err := validate.GetFreeEmailList("./free_emails.toml")
	if err != nil {
		log.Fatal(err)
	}

	roleAccounts, err := validate.GetRoleAccounts("./role_emails.toml")
	if err != nil {
		log.Fatal(err)
	}

	email := parseArgs()

	// build validation request
	request := validate.EmailValidationRequest{
		Email:            email,
		FromDomain:       "hubspot.com",
		FromEmail:        "yamini.rangan@hubspot.com",
		CatchAllTestUser: generateCatchAllUsername(),
	}

	syntaxResults := validate.ValidateEmailSyntax(email)
	domainResults := validate.ValidateDomain(request, knownProviders, true)
	emailResults := validate.ValidateEmail(request, knownProviders, freeEmails, roleAccounts)
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

func generateCatchAllUsername() string {
	rng, err := codename.DefaultRNG()
	if err != nil {
		panic(err)
	}
	name := codename.Generate(rng, 0)
	return strings.ReplaceAll(name, "-", "")
}

func buildResponse(syntax validate.SyntaxValidation, domain validate.DomainValidation, email validate.EmailValidatation) {
	isRisky := false
	if email.IsFreeAccount || email.IsRoleAccount || email.IsMailboxFull || domain.IsCatchAll || domain.IsFirewalled {
		isRisky = true
	}

	fmt.Println("isDeliverable:", email.IsDeliverable)
	fmt.Println("isValidSyntax:", syntax.IsValid)
	fmt.Println("provider:", domain.Provider)
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
	fmt.Println("Finance:", domain.AuthorizedSenders.Finance)
	fmt.Println("Hosting:", domain.AuthorizedSenders.Hosting)
	fmt.Println("Marketing:", domain.AuthorizedSenders.Marketing)
	fmt.Println("Security:", domain.AuthorizedSenders.Security)
	fmt.Println("Support:", domain.AuthorizedSenders.Support)
	fmt.Println("Transactional:", domain.AuthorizedSenders.Transactional)
	fmt.Println("Webmail:", domain.AuthorizedSenders.Webmail)
	fmt.Println("")
	fmt.Println("SMTP Errors")
	fmt.Println(email.SmtpError)
}
