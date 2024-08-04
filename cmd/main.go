package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/customeros/mailhawk/internal/dns"
	"github.com/customeros/mailhawk/internal/validate"
)

func main() {
	knownProviders, freeEmails, roleAccounts := getConfig()

	email := parseArgs()

	// build validation request
	request := validate.EmailValidationRequest{
		Email:            email,
		FromDomain:       "hubspot.com",
		FromEmail:        "harry@hubspot.com",
		CatchAllTestUser: "blueelephantpurpledinosaur",
	}

	syntaxResults := validate.ValidateEmailSyntax(email)
	domainResults := validate.ValidateDomain(request, knownProviders)
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

func getConfig() (dns.KnownProviders, validate.FreeEmails, validate.RoleAccounts) {
	knownProvidersPath, err := getConfigPath("known_email_providers.toml")
	if err != nil {
		log.Fatal(err)
	}

	knownProviders, err := dns.GetKnownProviders(knownProvidersPath)
	if err != nil {
		log.Fatal(err)
	}

	freeEmailsPath, err := getConfigPath("free_emails.toml")
	if err != nil {
		log.Fatal(err)
	}

	freeEmails, err := validate.GetFreeEmailList(freeEmailsPath)
	if err != nil {
		log.Fatal(err)
	}

	roleAccountsPath, err := getConfigPath("role_emails.toml")
	if err != nil {
		log.Fatal(err)
	}

	roleAccounts, err := validate.GetRoleAccounts(roleAccountsPath)
	if err != nil {
		log.Fatal(err)
	}
	return knownProviders, freeEmails, roleAccounts
}

func getConfigPath(configFilename string) (string, error) {
	// Get the directory of the current file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("error getting current file path")
	}

	// Navigate up to the project root (where go.mod is located)
	projectRoot := filepath.Dir(filepath.Dir(filename))

	// Construct the path to the config file
	configPath := filepath.Join(projectRoot, configFilename)

	return configPath, nil
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
