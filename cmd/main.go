package main

import (
	"fmt"
	"os"

	"github.com/customeros/mailhawk/internal/mx"
	smpt "github.com/customeros/mailhawk/internal/smtp"
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
	fmt.Println("Firewall:", mx.IsFirewall(mx.GetEmailServiceProviderFromMX(mxRecord)))
	esp, _ := mx.GetEmailProvidersFromSPF(email)
	fmt.Println("ESPs:", esp)

	proxy := smpt.ProxySetup{
		Enable:   false,
		Address:  "212.116.243.131:12323",
		Username: "14a8e1626bbd4",
		Password: "95f42a5e9c",
	}

	verified, err := smpt.VerifyEmailAddress(email, "microsoft.com", "steve.balmer@microsoft.com", proxy)
	fmt.Println("Verified:", verified)
	fmt.Println(err)
}
