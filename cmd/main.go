package main

import (
	"fmt"
	"os"

	"github.com/customeros/mailhawk/internal/checks"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <email>")
		return
	}
	email := os.Args[1]
	freeList := "/Users/mbrown/src/github.com/customeros/mailhawk/free_emails.toml"
	fmt.Println(checks.IsFreeEmailCheck(email, freeList))
}
