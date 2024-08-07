package mailvalidate

import (
	"embed"
	"fmt"
	"slices"

	"github.com/BurntSushi/toml"

	"github.com/customeros/mailsherpa/internal/syntax"
)

//go:embed free_emails.toml
var freeEmailsFile embed.FS

type FreeEmails struct {
	FreeEmailList []string `toml:"free_emails"`
}

func IsFreeEmailCheck(email string, freeEmails FreeEmails) (bool, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return false, fmt.Errorf("Not a valid email address")
	}

	if slices.Contains(freeEmails.FreeEmailList, domain) {
		return true, nil
	}
	return false, nil
}

func GetFreeEmailList() (FreeEmails, error) {
	var freeEmails FreeEmails

	// Read the file
	fileData, err := freeEmailsFile.ReadFile("free_emails.toml")
	if err != nil {
		return FreeEmails{}, err
	}

	if _, err := toml.Decode(string(fileData), &freeEmails); err != nil {
		return FreeEmails{}, fmt.Errorf("failed to decode TOML: %w", err)
	}

	return freeEmails, nil
}
