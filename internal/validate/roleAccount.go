package validate

import (
	"fmt"
	"os"
	"slices"

	"github.com/BurntSushi/toml"

	"github.com/customeros/mailhawk/internal/syntax"
)

type RoleAccounts struct {
	RoleAccountList []string `toml:"role_emails"`
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
