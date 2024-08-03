package checks

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

type RoleAccounts struct {
	RoleAccountList []string `toml:"role_emails"`
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

func getRoleAccounts(roleAccountFilePath string) (RoleAccounts, error) {
	var roleAccounts RoleAccounts
	content, err := os.ReadFile(roleAccountFilePath)
	if err != nil {
		return RoleAccounts{}, fmt.Errorf("failed to read role_account file: %w", err)
	}

	if _, err := toml.Decode(string(content), &roleAccounts); err != nil {
		return RoleAccounts{}, fmt.Errorf("failed to decode TOML: %w", err)
	}

	return roleAccounts, nil
}

func IsRoleAccountCheck(email, roleAccountFilePath string) (bool, error) {
	user, _, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return false, fmt.Errorf("Not a valid email address")
	}

	roleAccounts, err := getRoleAccounts(roleAccountFilePath)
	if err != nil {
		return false, err
	}
	if slices.Contains(roleAccounts.RoleAccountList, user) {
		return true, nil
	}
	return false, nil
}
