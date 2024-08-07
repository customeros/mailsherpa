package datastudy

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/lucasepe/codename"

	"github.com/customeros/mailsherpa/internal/dns"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type DomainResponse struct {
	email         string
	isDeliverable bool
	isSyntaxValid bool
	provider      string
	isRisky       bool
	isFirewalled  bool
	isFreeAccount bool
	isRoleAccount bool
	isMailboxFull bool
	isCatchAll    bool
	smtpError     string
}

func read_csv(filePath string) ([]string, error) {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all records from the CSV
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	// Check if the CSV has at least one row (header) and one column
	if len(records) < 1 || len(records[0]) != 1 {
		return nil, fmt.Errorf("invalid CSV format: expected 1 column with a header row")
	}

	// Create a slice to store the data (excluding the header)
	data := make([]string, 0, len(records)-1)

	// Append each row (skipping the header) to the data slice
	for _, row := range records[1:] {
		data = append(data, row[0])
	}

	return data, nil
}

func writeResultsFile(results []DomainResponse, filePath string) error {
	// Create or open the CSV file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Create a new CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header row
	header := []string{"Email", "IsDeliverable", "IsSyntaxValid", "Provider", "IsRisky", "IsFirewalled", "IsFreeAccount", "IsRoleAccount", "IsMailboxFull", "IsCatchAll", "SMTPError"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}

	// Write each DomainResponse as a row in the CSV
	for _, resp := range results {
		row := []string{
			resp.email,
			strconv.FormatBool(resp.isDeliverable),
			strconv.FormatBool(resp.isSyntaxValid),
			resp.provider,
			strconv.FormatBool(resp.isRisky),
			strconv.FormatBool(resp.isFirewalled),
			strconv.FormatBool(resp.isFreeAccount),
			strconv.FormatBool(resp.isRoleAccount),
			strconv.FormatBool(resp.isMailboxFull),
			strconv.FormatBool(resp.isCatchAll),
			resp.smtpError,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %w", err)
		}
	}

	return nil
}

func RunDataStudy(inputFilePath, outputFilePath string) {
	knownProviders, err := dns.GetKnownProviders("./known_email_providers.toml")
	if err != nil {
		log.Fatal(err)
	}

	freeEmails, err := mailvalidate.GetFreeEmailList("./free_emails.toml")
	if err != nil {
		log.Fatal(err)
	}

	roleAccounts, err := mailvalidate.GetRoleAccounts("./role_emails.toml")
	if err != nil {
		log.Fatal(err)
	}

	testEmails, err := read_csv(inputFilePath)
	if err != nil {
		log.Fatal(err)
	}

	var output []DomainResponse
	catchAllResults := make(map[string]bool)

	for v, email := range testEmails {
		fmt.Println(v)

		request := mailvalidate.EmailValidationRequest{
			Email:            email,
			FromDomain:       "hubspot.com",
			FromEmail:        "yamini.rangan@hubspot.com",
			CatchAllTestUser: generateCatchAllUsername(),
		}

		_, domain, _ := syntax.GetEmailUserAndDomain(email)
		validateCatchAll := false
		if _, exists := catchAllResults[domain]; !exists {
			validateCatchAll = true
		}

		syntaxResults := mailvalidate.ValidateEmailSyntax(email)
		domainResults := mailvalidate.ValidateDomain(request, knownProviders, validateCatchAll)
		emailResults := mailvalidate.ValidateEmail(request, knownProviders, freeEmails, roleAccounts)

		isRisky := false
		if emailResults.IsFreeAccount || emailResults.IsRoleAccount || emailResults.IsMailboxFull || domainResults.IsCatchAll || domainResults.IsFirewalled {
			isRisky = true
		}

		isCatchAll := domainResults.IsCatchAll
		if validateCatchAll {
			catchAllResults[domain] = isCatchAll
		} else {
			isCatchAll = catchAllResults[domain]
		}

		results := DomainResponse{
			email:         email,
			isDeliverable: emailResults.IsDeliverable,
			isSyntaxValid: syntaxResults.IsValid,
			provider:      domainResults.Provider,
			isRisky:       isRisky,
			isFirewalled:  domainResults.IsFirewalled,
			isFreeAccount: emailResults.IsFreeAccount,
			isRoleAccount: emailResults.IsRoleAccount,
			isMailboxFull: emailResults.IsMailboxFull,
			isCatchAll:    isCatchAll,
			smtpError:     emailResults.SmtpError,
		}
		output = append(output, results)
	}
	writeResultsFile(output, outputFilePath)
}

func generateCatchAllUsername() string {
	rng, err := codename.DefaultRNG()
	if err != nil {
		panic(err)
	}
	name := codename.Generate(rng, 0)
	return strings.ReplaceAll(name, "-", "")
}
