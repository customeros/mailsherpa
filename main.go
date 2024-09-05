package main

import (
	"flag"
	"fmt"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/cmd"
)

func main() {
	flag.Parse()
	args := flag.Args()

	switch args[0] {
	case "domain":
		if len(args) != 2 {
			fmt.Println("Usage: mailsherpa domain <domain>")
			return
		}
		cmd.VerifyDomain(args[1], true)
	case "syntax":
		if len(args) != 2 {
			fmt.Println("Usage: mailsherpa syntax <email>")
			return
		}
		cmd.VerifySyntax(args[1], true)
	case "redirect":
		fmt.Println(domaincheck.PrimaryDomainCheck(args[1]))
	case "version":
		cmd.Version()
	default:
		if len(args) < 1 {
			cmd.PrintUsage()
			return
		}
		cmd.VerifyEmail(args[0])
	}
}
