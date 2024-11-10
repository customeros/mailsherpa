package main

import (
	"flag"
	"fmt"

	"github.com/customeros/mailsherpa/cli"
	"github.com/customeros/mailsherpa/domaincheck"
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
		cli.VerifyDomain(args[1], true)
	case "syntax":
		if len(args) != 2 {
			fmt.Println("Usage: mailsherpa syntax <email>")
			return
		}
		cli.VerifySyntax(args[1], true)
	case "redirect":
		fmt.Println(domaincheck.PrimaryDomainCheck(args[1]))
	case "version":
		cli.Version()
	default:
		if len(args) < 1 {
			cli.PrintUsage()
			return
		}
		cli.VerifyEmail(args[0])
	}
}
