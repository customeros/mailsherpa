package names

import (
	"bufio"
	"embed"
	"strings"
)

//go:embed first_names.txt surnames.txt
var namesFS embed.FS

// LoadFirstNames reads first names from the embedded file and returns them as a map[string]bool.
func LoadFirstNames() (map[string]bool, error) {
	names := make(map[string]bool)

	file, err := namesFS.Open("first_names.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			names[strings.ToLower(name)] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return names, nil
}

// LoadSurnames reads surnames from the embedded file and returns them as a map[string]bool.
func LoadSurnames() (map[string]bool, error) {
	names := make(map[string]bool)

	file, err := namesFS.Open("surnames.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			names[strings.ToLower(name)] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return names, nil
}

