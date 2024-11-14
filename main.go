package main

import (
	"flag"
	"fmt"

	"github.com/customeros/mailsherpa/cli"
	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/domaingen"
	"github.com/customeros/mailsherpa/emailparser"
)

func main() {
	flag.Parse()
	args := flag.Args()

	switch args[0] {
	case "avail":
		_, avail := domaingen.IsDomainAvailable(args[1])
		fmt.Println("Domain Available:", avail)
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
	case "rec":
		fmt.Println(domaingen.RecommendOutboundDomains(args[1], 20))
	case "redirect":
		fmt.Println(domaincheck.PrimaryDomainCheck(args[1]))
	case "parse":
		fmt.Println(emailparser.Parse(args[1]))
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
