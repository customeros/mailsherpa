package bulkvalidate

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/lucasepe/codename"
	"github.com/schollz/progressbar/v3"

	"github.com/customeros/mailsherpa/internal/run"
	"github.com/customeros/mailsherpa/internal/syntax"
	"github.com/customeros/mailsherpa/mailvalidate"
)

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

func writeResultsFile(results []run.VerifyEmailResponse, filePath string) error {
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
	header := []string{
		"Email",
		"Username",
		"Domain",
		"IsValidSyntax",
		"IsDeliverable",
		"Provider",
		"Firewall",
		"IsRisky",
		"IsFirewalled",
		"IsFreeAccount",
		"IsRoleAccount",
		"IsMailboxFull",
		"IsCatchAll",
		"SmtpSuccess",
		"SmtpRetry",
		"SmtpResponseCode",
		"SmtpErrorCode",
		"SmtpDescription",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}

	// Write each DomainResponse as a row in the CSV
	for _, resp := range results {
		row := []string{
			resp.Email,
			resp.Syntax.User,
			resp.Syntax.Domain,
			strconv.FormatBool(resp.Syntax.IsValid),
			strconv.FormatBool(resp.IsDeliverable),
			resp.Provider,
			resp.Firewall,
			strconv.FormatBool(resp.IsRisky),
			strconv.FormatBool(resp.Risk.IsFirewalled),
			strconv.FormatBool(resp.Risk.IsFreeAccount),
			strconv.FormatBool(resp.Risk.IsRoleAccount),
			strconv.FormatBool(resp.Risk.IsMailboxFull),
			strconv.FormatBool(resp.Risk.IsCatchAll),
			strconv.FormatBool(resp.Smtp.Success),
			strconv.FormatBool(resp.Smtp.Retry),
			resp.Smtp.ResponseCode,
			resp.Smtp.ErrorCode,
			resp.Smtp.Description,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %w", err)
		}
	}

	return nil
}

func RunBulkValidation(inputFilePath, outputFilePath string) error {
	testEmails, err := read_csv(inputFilePath)
	if err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	catchAllResults := make(map[string]bool)
	var output []run.VerifyEmailResponse

	bar := progressbar.Default(int64(len(testEmails)))

	for _, email := range testEmails {
		bar.Add(1)
		request := run.BuildRequest(email)

		_, domain, _ := syntax.GetEmailUserAndDomain(email)
		validateCatchAll := false
		if _, exists := catchAllResults[domain]; !exists {
			validateCatchAll = true
		}

		syntaxResults := mailvalidate.ValidateEmailSyntax(email)
		domainResults, err := mailvalidate.ValidateDomain(request, validateCatchAll)
		if err != nil {
			log.Printf("Error: %s %s", email, err.Error())
		}
		emailResults, err := mailvalidate.ValidateEmail(request)
		if err != nil {
			log.Printf("Error: %s %s", email, err.Error())
		}

		isCatchAll := domainResults.IsCatchAll
		if validateCatchAll {
			catchAllResults[domain] = isCatchAll
		} else {
			isCatchAll = catchAllResults[domain]
		}

		results := run.BuildResponse(email, syntaxResults, domainResults, emailResults)
		output = append(output, results)
	}
	writeResultsFile(output, outputFilePath)

	return nil
}

func generateCatchAllUsername() string {
	rng, err := codename.DefaultRNG()
	if err != nil {
		panic(err)
	}
	name := codename.Generate(rng, 0)
	return strings.ReplaceAll(name, "-", "")
}
