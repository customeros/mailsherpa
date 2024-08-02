package main

import (
	"fmt"
	"os"

	"github.com/customeros/mailhawk/internal/mx"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <email>")
		return
	}
	email := os.Args[1]

	mxRecord, _ := mx.GetMXRecordsForEmail(email)
	fmt.Println(mxRecord)
	fmt.Println(mx.GetEmailServiceProviderFromMX(mxRecord))
}
