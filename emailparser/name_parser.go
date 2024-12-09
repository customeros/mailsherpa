package emailparser

import (
	"errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"regexp"
	"strings"

	"github.com/customeros/mailsherpa/internal/names"
	"github.com/customeros/mailsherpa/internal/syntax"
)

type ParsedEmail struct {
	Email     string
	FirstName string
	LastName  string
	Pattern   string
}

type Pattern string

const (
	PatternDelimited      Pattern = "delimited"       // brugnone.fabio
	PatternFirstName      Pattern = "firstname"       // abigail
	PatternCombined       Pattern = "combined"        // adambangh, michaelstewart
	PatternFullName       Pattern = "fullname"        // joshua.j.kim, brownjasmine
	PatternNameInitial    Pattern = "name.initial"    // tyler.g, colinj
	PatternInitialSurname Pattern = "initial.surname" // pslack, nfalletti
	PatternSurnameInitial Pattern = "surname.initial" // chapmann.a
	PatternUnknown        Pattern = "unknown"
)

var (
	ErrInvalidEmail = errors.New("invalid email address")
	ErrNoName       = errors.New("could not extract name from email")

	titleCaser = cases.Title(language.English)
	firstNames map[string]bool
	surnames   map[string]bool
)

func Parse(email string) (ParsedEmail, error) {
	ok, cleanEmail, username, _ := syntax.NormalizeEmailAddress(email)
	if !ok {
		return ParsedEmail{}, ErrInvalidEmail
	}

	username = strings.ToLower(username)

	// Try delimited format first if it contains a dot
	if strings.Contains(username, ".") {
		if result, ok := tryDelimitedFormat(username); ok {
			result.Email = cleanEmail
			return result, nil
		}

		// If delimited format didn't match, remove dots before trying other patterns
		username = strings.ReplaceAll(username, ".", "")
	}

	// Try other patterns with cleaned username
	for _, tryPattern := range []func(string) (ParsedEmail, bool){
		trySingleName,
		tryCombinedName,
		tryNameWithInitial,
		tryInitialSurname,
	} {
		if result, ok := tryPattern(username); ok {
			result.Email = cleanEmail
			return result, nil
		}
	}

	return ParsedEmail{
		Email:   cleanEmail,
		Pattern: string(PatternUnknown),
	}, nil
}

// tryDelimitedFormat handles all patterns with dots
func tryDelimitedFormat(username string) (ParsedEmail, bool) {
	parts := strings.Split(username, ".")

	// Handle two-part names (first.last or surname.firstname)
	if len(parts) == 2 {
		first := cleanString(parts[0])
		second := cleanString(parts[1])

		// Check for firstname.last_initial pattern (e.g., tyler.g)
		if len(second) == 1 && isLikelyFirstName(first) {
			return ParsedEmail{
				FirstName: titleCaser.String(first),
				LastName:  strings.ToUpper(second),
				Pattern:   string(PatternNameInitial),
			}, true
		}

		// Check for surname.initial pattern (e.g., chapmann.a)
		if len(second) == 1 && (isLikelySurname(first) || len(first) >= 4) {
			return ParsedEmail{
				FirstName: strings.ToUpper(second),
				LastName:  titleCaser.String(first),
				Pattern:   string(PatternSurnameInitial),
			}, true
		}

		// Check for surname.firstname pattern (e.g., brugnone.fabio)
		if isLikelySurname(first) && isLikelyFirstName(second) {
			return ParsedEmail{
				FirstName: titleCaser.String(second),
				LastName:  titleCaser.String(first),
				Pattern:   string(PatternDelimited),
			}, true
		}
	}

	// Handle three-part names (e.g., joshua.j.kim)
	if len(parts) == 3 {
		first := cleanString(parts[0])
		last := cleanString(parts[2])

		if isLikelyFirstName(first) {
			return ParsedEmail{
				FirstName: titleCaser.String(first),
				LastName:  titleCaser.String(last),
				Pattern:   string(PatternFullName),
			}, true
		}
	}

	return ParsedEmail{}, false
}

// tryNameWithInitial handles trailing initials without dots
func tryNameWithInitial(username string) (ParsedEmail, bool) {
	if len(username) < 5 {
		return ParsedEmail{}, false
	}
	// Try to find a known first name
	for i := 2; i < len(username); i++ {
		possibleName := username[:i]
		possibleInitial := username[i:]

		if len(possibleInitial) == 1 && isLikelyFirstName(possibleName) {
			return ParsedEmail{
				FirstName: titleCaser.String(possibleName),
				LastName:  strings.ToUpper(possibleInitial),
				Pattern:   string(PatternNameInitial),
			}, true
		}
	}
	return ParsedEmail{}, false
}

// tryInitialSurname handles initial+surname without dots
func tryInitialSurname(username string) (ParsedEmail, bool) {
	if len(username) > 2 {
		initial := username[0:1]
		surname := username[1:]

		if isLikelySurname(surname) {
			return ParsedEmail{
				FirstName: strings.ToUpper(initial),
				LastName:  titleCaser.String(surname),
				Pattern:   string(PatternInitialSurname),
			}, true
		}
	}
	return ParsedEmail{}, false
}

// tryCombinedName handles combined names by looking for known first names
func tryCombinedName(username string) (ParsedEmail, bool) {
	username = cleanString(username)
	// Only proceed if username is long enough
	if len(username) < 5 {
		return ParsedEmail{}, false
	}

	// Track best match for first and last names based on known first names
	bestLen := 0
	var bestFirst, bestLast string

	// Iterate to find the longest possible first name prefix
	for i := 2; i < len(username)-2; i++ {
		possible := username[:i]
		remaining := username[i:]

		if isLikelyFirstName(possible) && isLikelySurname(remaining) {
			bestFirst = possible
			bestLast = remaining
			bestLen = len(possible)
			break
		}
	}

	if bestLen > 0 {
		return ParsedEmail{
			FirstName: titleCaser.String(bestFirst),
			LastName:  titleCaser.String(bestLast),
			Pattern:   string(PatternCombined),
		}, true
	}

	return ParsedEmail{}, false
}

// trySingleName handles single name cases
func trySingleName(username string) (ParsedEmail, bool) {
	if isLikelyFirstName(username) {
		return ParsedEmail{
			FirstName: titleCaser.String(username),
			Pattern:   string(PatternFirstName),
		}, true
	}
	return ParsedEmail{}, false
}

func init() {
	var err error
	firstNames, err = names.LoadFirstNames()
	if err != nil {
		log.Fatalf("Failed to load first names: %v", err)
	}

	surnames, err = names.LoadSurnames()
	if err != nil {
		log.Fatalf("Failed to load surnames: %v", err)
	}
}

func isLikelyFirstName(name string) bool {
	cleanName := cleanString(name)
	return firstNames[strings.ToLower(cleanName)]
}

func isLikelySurname(name string) bool {
	cleanName := cleanString(name)
	return surnames[strings.ToLower(cleanName)]
}

func cleanString(s string) string {
	// Remove numbers first
	s = regexp.MustCompile(`[0-9]`).ReplaceAllString(s, "")

	// Convert to lowercase
	s = strings.ToLower(s)

	// Remove everything except letters
	s = regexp.MustCompile(`[^a-z]`).ReplaceAllString(s, "")

	return s
}
