package roleaccounts

import (
	"embed"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/exp/slices"
)

//go:embed role_emails.toml
var roleEmailsFile embed.FS

type RoleAccounts struct {
	Contains []string `toml:"contains"`
	Matches  []string `toml:"matches"`
}

func IsRoleAccountCheck(username string) (bool, error) {
	roleAccounts, err := getRoleAccounts()
	if err != nil {
		return false, err
	}

	if slices.Contains(roleAccounts.Matches, username) {
		return true, nil
	}

	for _, value := range roleAccounts.Contains {
		if strings.Contains(username, value) {
			return true, nil
		}
	}

	return false, nil
}

func getRoleAccounts() (RoleAccounts, error) {
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
