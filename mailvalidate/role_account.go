package mailvalidate

import (
	"embed"
	"fmt"

	"github.com/BurntSushi/toml"
	"golang.org/x/exp/slices"

	"github.com/customeros/mailsherpa/internal/syntax"
)

//go:embed role_emails.toml
var roleEmailsFile embed.FS

type RoleAccounts struct {
	RoleAccountList []string `toml:"role_emails"`
}

func IsRoleAccountCheck(email string, roleAccounts *RoleAccounts) (bool, error) {
	user, _, ok := syntax.GetEmailUserAndDomain(email)
	if !ok {
		return false, fmt.Errorf("Not a valid email address")
	}

	if slices.Contains(roleAccounts.RoleAccountList, user) {
		return true, nil
	}
	return false, nil
}

func GetRoleAccounts() (RoleAccounts, error) {
	var roleAccounts RoleAccounts

	// Read the file
	fileData, err := roleEmailsFile.ReadFile("role_emails.toml")
	if err != nil {
		return RoleAccounts{}, err
	}

	if _, err := toml.Decode(string(fileData), &roleAccounts); err != nil {
		return RoleAccounts{}, fmt.Errorf("failed to decode TOML: %w", err)
	}

	return roleAccounts, nil
}
