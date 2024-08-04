package dns

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Provider struct {
	SPF  string `toml:"spf"`
	Name string `toml:"name"`
	Type string `toml:"type"`
}

type KnownProviders map[string]Provider

func GetKnownProviders(filename string) (KnownProviders, error) {
	var domain KnownProviders

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Decode the TOML content
	if err := toml.Unmarshal(content, &domain); err != nil {
		return nil, fmt.Errorf("error decoding TOML: %w", err)
	}

	return domain, nil
}