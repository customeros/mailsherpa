package main

import (
	"fmt"
	"os"

	"github.com/customeros/mailhawk/internal/dns"
)

type Config struct {
	freeEmailsFile     string
	roleAccountsFile   string
	emailProvidersFile string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <email>")
		return
	}
	email := os.Args[1]
	freeList := "/Users/mbrown/src/github.com/customeros/mailhawk/known_email_providers.toml"
	senders, err := dns.GetAuthorizedSenders(email, freeList)
	fmt.Println(err)
	fmt.Println(senders)
}
