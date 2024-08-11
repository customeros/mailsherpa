package main

import (
	"flag"
	"fmt"

	"github.com/customeros/mailsherpa/internal/cmd"
)

func main() {
	flag.Parse()
	args := flag.Args()

	switch args[0] {
	case "bulk":
		if len(args) != 3 {
			fmt.Println("Usage: mailsherpa bulk <input file> <output file>")
			return
		}
		cmd.BulkVerify(args[1], args[2])
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
