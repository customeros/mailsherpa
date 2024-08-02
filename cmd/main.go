package main

import (
	"fmt"
	emailSyntax "github.com/customeros/mailhawk/internal/syntax"
)

func main() {
	testEmails := []string{
		"user@example.com",
		"user.name+tag@example.com",
		"user.name@example.co.uk",
		"user@localhost",
		"user@192.168.1.1",
		"user.name@example..com",
		"user@.com",
		"@example.com",
		"user@example.",
		"user name@example.com",
	}

	for _, email := range testEmails {
		fmt.Printf("%-30s : %v\n", email, emailSyntax.IsValidEmailSyntax(email))
	}
}
