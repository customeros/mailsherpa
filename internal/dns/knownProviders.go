package dns

import (
	"embed"
	"fmt"
	"github.com/BurntSushi/toml"
)

//go:embed known_email_providers.toml
var knownProvidersFile embed.FS

type ProviderCategory struct {
	Type    string     `toml:"type"`
	Domains [][]string `toml:"domains"`
}

type KnownProviders struct {
	Enterprise ProviderCategory `toml:"enterprise"`
	Hosting    ProviderCategory `toml:"hosting"`
	Webmail    ProviderCategory `toml:"webmail"`
	Security   ProviderCategory `toml:"security"`
}

func GetKnownProviders() (*KnownProviders, error) {
	var providers KnownProviders

	// Read the file
	fileData, err := knownProvidersFile.ReadFile("known_email_providers.toml")
	if err != nil {
		return nil, err
	}

	// Decode the TOML content
	if err := toml.Unmarshal(fileData, &providers); err != nil {
		return nil, fmt.Errorf("error decoding TOML: %w", err)
	}

	// Set the Type field for each category
	providers.Enterprise.Type = "enterprise"
	providers.Hosting.Type = "hosting"
	providers.Webmail.Type = "webmail"
	providers.Security.Type = "security"

	return &providers, nil
}

// Helper function to get provider by domain
func (kp *KnownProviders) GetProviderByDomain(domain string) (string, string) {
	categories := []ProviderCategory{
		kp.Enterprise,
		kp.Hosting,
		kp.Webmail,
		kp.Security,
	}

	for _, category := range categories {
		for _, provider := range category.Domains {
			if provider[0] == domain {
				return provider[1], category.Type
			}
		}
	}

	return "", ""
}
