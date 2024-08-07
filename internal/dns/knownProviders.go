package dns

import (
	"embed"
	"fmt"
	"github.com/BurntSushi/toml"
)

//go:embed known_email_providers.toml
var knownProvidersFile embed.FS

type Provider struct {
	SPF  string `toml:"spf"`
	Name string `toml:"name"`
	Type string `toml:"type"`
}

type KnownProviders map[string]Provider

func GetKnownProviders() (KnownProviders, error) {
	var domain KnownProviders

	// Read the file
	fileData, err := knownProvidersFile.ReadFile("known_email_providers.toml")
	if err != nil {
		return nil, err
	}

	// Decode the TOML content
	if err := toml.Unmarshal(fileData, &domain); err != nil {
		return nil, fmt.Errorf("error decoding TOML: %w", err)
	}

	return domain, nil
}
