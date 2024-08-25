package bulkvalidate

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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

const (
	batchSize      = 10
	checkpointFile = "validation_checkpoint.json"
)

type Checkpoint struct {
	ProcessedRows int `json:"processedRows"`
}

func RunBulkValidation(inputFilePath, outputFilePath string) error {
	checkpoint, err := loadCheckpoint()
	if err != nil {
		return fmt.Errorf("error loading checkpoint: %w", err)
	}

	reader, file, err := read_csv(inputFilePath)
	if err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}
	defer file.Close()

	// Read and store the header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("error reading header: %w", err)
	}

	// Skip to the last processed row
	for i := 0; i < checkpoint.ProcessedRows; i++ {
		_, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error skipping to checkpoint: %w", err)
		}
	}

	catchAllResults := make(map[string]bool)
	bar := progressbar.Default(-1)

	outputFileExists := fileExists(outputFilePath)

	for {
		batch, err := readBatch(reader, batchSize)
		if err != nil {
			return fmt.Errorf("error reading batch: %w", err)
		}
		if len(batch) == 0 {
			break
		}

		results := processBatch(batch, catchAllResults)

		err = writeResultsFile(results, outputFilePath, outputFileExists, header)
		if err != nil {
			return fmt.Errorf("error writing results: %w", err)
		}

		checkpoint.ProcessedRows += len(batch)
		err = saveCheckpoint(checkpoint)
		if err != nil {
			return fmt.Errorf("error saving checkpoint: %w", err)
		}

		bar.Add(len(batch))
		outputFileExists = true
	}

	return nil
}

func read_csv(filePath string) (*csv.Reader, *os.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}

	reader := csv.NewReader(bufio.NewReader(file))
	return reader, file, nil
}

func writeResultsFile(results []run.VerifyEmailResponse, filePath string, append bool, header []string) error {
	flag := os.O_CREATE | os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !append {
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("error writing header: %w", err)
		}
	}

	for _, resp := range results {
		row := []string{
			resp.Email, resp.Syntax.User, resp.Syntax.Domain,
			strconv.FormatBool(resp.Syntax.IsValid),
			resp.IsDeliverable,
			resp.Provider, resp.Firewall,
			strconv.FormatBool(resp.IsRisky),
			strconv.FormatBool(resp.Risk.IsFirewalled),
			strconv.FormatBool(resp.Risk.IsFreeAccount),
			strconv.FormatBool(resp.Risk.IsRoleAccount),
			strconv.FormatBool(resp.Risk.IsMailboxFull),
			strconv.FormatBool(resp.IsCatchAll),
			resp.Smtp.ResponseCode, resp.Smtp.ErrorCode, resp.Smtp.Description,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %w", err)
		}
	}

	return nil
}

func readBatch(reader *csv.Reader, batchSize int) ([]string, error) {
	var batch []string
	for i := 0; i < batchSize; i++ {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				// End of file reached
				return batch, nil
			}
			if err == csv.ErrFieldCount {
				// Skip records with incorrect field count
				continue
			}
			// Return any other error
			return batch, err
		}
		if len(record) > 0 {
			batch = append(batch, record[0])
		}
	}
	return batch, nil
}

func processBatch(batch []string, catchAllResults map[string]bool) []run.VerifyEmailResponse {
	var results []run.VerifyEmailResponse

	for _, email := range batch {
		request := run.BuildRequest(email)
		_, domain, _ := syntax.GetEmailUserAndDomain(email)
		validateCatchAll := false
		if _, exists := catchAllResults[domain]; !exists {
			validateCatchAll = true
		}
		syntaxResults := mailvalidate.ValidateEmailSyntax(email)
		domainResults := mailvalidate.ValidateDomain(request)
		if domainResults.Error != "" {
			log.Println(domainResults.Error)
		}
		emailResults := mailvalidate.ValidateEmail(request)
		if emailResults.Error != "" {
			log.Println(domainResults.Error)
		}
		isCatchAll := domainResults.IsCatchAll
		if validateCatchAll {
			catchAllResults[domain] = isCatchAll
		}
		result := run.BuildResponse(email, syntaxResults, domainResults, emailResults)
		results = append(results, result)
	}

	return results
}

func loadCheckpoint() (Checkpoint, error) {
	var checkpoint Checkpoint
	file, err := os.Open(checkpointFile)
	if os.IsNotExist(err) {
		return checkpoint, nil
	}
	if err != nil {
		return checkpoint, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&checkpoint)
	return checkpoint, err
}

func saveCheckpoint(checkpoint Checkpoint) error {
	file, err := os.Create(checkpointFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(checkpoint)
}

func generateCatchAllUsername() string {
	rng, err := codename.DefaultRNG()
	if err != nil {
		panic(err)
	}
	name := codename.Generate(rng, 0)
	return strings.ReplaceAll(name, "-", "")
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
