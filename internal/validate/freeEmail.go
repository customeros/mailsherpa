package validate

import (
	"fmt"
	"os"
	"slices"

	"github.com/BurntSushi/toml"

	"github.com/customeros/mailhawk/internal/syntax"
)

type FreeEmails struct {
	FreeEmailList []string `toml:"free_emails"`
}

func IsFreeEmailCheck(email, freeEmailFilePath string) (bool, error) {
	_, domain, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return false, fmt.Errorf("Not a valid email address")
	}

	freeEmails, err := getFreeEmailList(freeEmailFilePath)
	if err != nil {
		return false, err
	}
	if slices.Contains(freeEmails.FreeEmailList, domain) {
		return true, nil
	}
	return false, nil
}

func getFreeEmailList(freeEmailFilePath string) (FreeEmails, error) {
	var freeEmails FreeEmails
	content, err := os.ReadFile(freeEmailFilePath)
	if err != nil {
		return FreeEmails{}, fmt.Errorf("failed to read free_email file: %w", err)
	}

	if _, err := toml.Decode(string(content), &freeEmails); err != nil {
		return FreeEmails{}, fmt.Errorf("failed to decode TOML: %w", err)
	}

	return freeEmails, nil
}
