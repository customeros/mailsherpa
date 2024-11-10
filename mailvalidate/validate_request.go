package mailvalidate

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/customeros/mailsherpa/domaincheck"
	"github.com/customeros/mailsherpa/internal/util"
)

type EmailValidationRequest struct {
	Email            string
	FromDomain       string
	FromEmail        string
	CatchAllTestUser string
	Dns              *domaincheck.DNS
	// applicable only for email validation. Pass results from domain validation
	DomainValidationParams *DomainValidationParams
}

func validateRequest(request *EmailValidationRequest) error {
	if request.Email == "" {
		return errors.New("Email is required")
	}
	if request.FromDomain == "" {
		return errors.New("FromDomain is required")
	}
	if request.FromEmail == "" {
		firstName, lastName := util.GenerateNames()
		request.FromEmail = fmt.Sprintf("%s.%s@%s", firstName, lastName, request.FromDomain)
	}
	if request.CatchAllTestUser == "" {
		request.CatchAllTestUser = util.GenerateCatchAllUsername()
	}
	return nil
}
