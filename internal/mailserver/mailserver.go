package mailserver

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/customeros/mailsherpa/domaincheck"
)

type SMPTValidation struct {
	CanConnectSmtp bool
	InboxFull      bool
	ResponseCode   string
	ErrorCode      string
	Description    string
	SmtpResponse   string
}

func VerifyEmailAddress(email, fromDomain, fromEmail string, dnsRecords domaincheck.DNS) SMPTValidation {
	results := SMPTValidation{}

	// Has MX Record Check
	if len(dnsRecords.MX) == 0 {
		results.CanConnectSmtp = false
		results.Description = "No MX records for domain"
		return results
	}

	var conn net.Conn
	var client *bufio.Reader
	var err error
	var greetCode string
	var greetDesc string

	for i := 0; i < len(dnsRecords.MX); i++ {
		conn, client, err = connectToSMTP(dnsRecords.MX[i])
		if err != nil {
			continue
		}
		greetCode, greetDesc = readSMTPgreeting(client)
		if greetCode == "220" {
			break
		}
	}

	if greetCode != "220" {
		results.CanConnectSmtp = false
		results.ResponseCode = greetCode
		results.Description = greetDesc
		if results.Description == "" {
			results.Description = "Cannot connect to any MX server"
		}
		return results
	}

	defer conn.Close()

	heloCode, heloDesc, heloErr := sendHELO(conn, client, fromDomain)
	if heloErr != nil {
		results.CanConnectSmtp = false
		log.Printf(heloErr.Error())
		return results
	}
	if heloCode != "250" {
		results.ResponseCode = heloCode
		results.Description = heloDesc
		results.CanConnectSmtp = false
		return results
	}

	fromCode, fromDesc, fromErr := sendMAILFROM(conn, client, fromEmail)
	if fromErr != nil {
		results.CanConnectSmtp = false
		log.Printf(fromErr.Error())
		return results
	}
	if fromCode != "250" {
		results.ResponseCode = fromCode
		results.Description = fromDesc
		results.CanConnectSmtp = false
		return results
	}

	results, err = sendRCPTTO(conn, client, email)
	if err != nil {
		results.CanConnectSmtp = false
		results.SmtpResponse = err.Error()
		return results
	}

	return results
}

func connectToSMTP(mxServer string) (net.Conn, *bufio.Reader, error) {
	conn, err := net.DialTimeout("tcp", mxServer+":25", 10*time.Second)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect to SMTP server")
	}

	client := bufio.NewReader(conn)
	return conn, client, nil
}

func readSMTPgreeting(smtpClient *bufio.Reader) (string, string) {
	var fullGreeting strings.Builder
	var code string

	for {
		line, err := smtpClient.ReadString('\n')
		if err != nil {
			return "", ""
		}

		// Trim the line to remove whitespace and newline characters
		line = strings.TrimSpace(line)

		// Append this line to the full greeting
		fullGreeting.WriteString(line + "\n")

		// Check if it's a continuation line
		if strings.HasPrefix(line, "220-") {
			continue
		} else if strings.HasPrefix(line, "220 ") {
			// This is the last line of the greeting
			code = "220"
			break
		} else {
			// Unexpected response
			parsedCode, _ := parseSmtpCommand(line)
			return parsedCode, fullGreeting.String()
		}
	}
	return code, fullGreeting.String()
}

func sendSMTPcommand(conn net.Conn, smtpClient *bufio.Reader, cmd string) (string, error) {
	_, err := fmt.Fprintf(conn, cmd+"\r\n")
	if err != nil {
		return "", fmt.Errorf("failed to send SMTP command %s: %s", cmd, err.Error())
	}
	resp, err := smtpClient.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read response for SMTP command %s: %s", cmd, err.Error())
	}
	return resp, nil
}

func sendHELO(conn net.Conn, smtpClient *bufio.Reader, fromDomain string) (string, string, error) {
	helo := fmt.Sprintf("HELO %s", fromDomain)
	resp, err := sendSMTPcommand(conn, smtpClient, helo)
	if err != nil {
		return "", "", fmt.Errorf("SMTP HELO command failed: %w", err)
	}
	statusCode, desc := parseSmtpCommand(resp)
	return statusCode, desc, nil
}

func sendMAILFROM(conn net.Conn, smtpClient *bufio.Reader, fromEmail string) (string, string, error) {
	mailfrom := fmt.Sprintf("MAIL FROM:<%s>", fromEmail)
	resp, err := sendSMTPcommand(conn, smtpClient, mailfrom)
	if err != nil {
		return "", "", fmt.Errorf("SMTP MAIL FROM command failed: %w", err)
	}
	statusCode, desc := parseSmtpCommand(resp)
	return statusCode, desc, nil
}

func sendRCPTTO(conn net.Conn, smtpClient *bufio.Reader, emailToValidate string) (results SMPTValidation, err error) {
	rcpt := fmt.Sprintf("RCPT TO:<%s>", emailToValidate)
	resp, err := sendSMTPcommand(conn, smtpClient, rcpt)
	if err != nil {
		return results, errors.Wrap(err, "RCPT TO command failed")
	}

	results.SmtpResponse = resp
	results.ResponseCode, results.ErrorCode, results.Description = ParseSmtpResponse(resp)

	if results.ResponseCode != "" {
		results.CanConnectSmtp = true
	}

	return
}

func ParseSmtpResponse(response string) (statusCode, errorCode, description string) {
	// Trim the input string
	response = strings.TrimSpace(response)

	// Extract the status code
	statusCodePattern := `^(\d{3})`
	statusCodeRegex := regexp.MustCompile(statusCodePattern)
	statusCodeMatch := statusCodeRegex.FindStringSubmatch(response)
	if len(statusCodeMatch) > 0 {
		statusCode = statusCodeMatch[1]
	} else {
		description = response
		return
	}

	// Extract the error code
	errorCodePattern := `\b(\d\.\d\.\d)\b`
	errorCodeRegex := regexp.MustCompile(errorCodePattern)
	errorCodeMatch := errorCodeRegex.FindStringSubmatch(response)
	if len(errorCodeMatch) > 0 {
		errorCode = errorCodeMatch[1]
	} else {
		errorCode = ""
	}

	// Extract the description
	if errorCode == "" {
		description = strings.TrimSpace(response[len(statusCode):])
	} else {
		startErrorCode := strings.Index(response, errorCode)
		endErrorCode := startErrorCode + len(errorCode)
		if startErrorCode <= 6 {
			description = strings.TrimSpace(response[endErrorCode:])
		} else {
			description = strings.TrimSpace(response[len(statusCode):])
		}
	}

	// Remove leading special characters from the description
	description = strings.TrimLeft(description, "]) #-}")

	return
}

func parseSmtpCommand(response string) (string, string) {
	// Check if the string is long enough to contain a status code
	if len(response) < 3 {
		return response, ""
	}

	// Extract the status code
	statusCode := response[:3]

	// Extract the rest of the message
	message := strings.TrimSpace(response[3:])
	message = strings.TrimPrefix(message, "-")

	return statusCode, message
}
