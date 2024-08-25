package mailserver

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/customeros/mailsherpa/internal/dns"
)

type SMPTValidation struct {
	CanConnectSmtp bool
	InboxFull      bool
	ResponseCode   string
	ErrorCode      string
	Description    string
	SmtpResponse   string
}

func VerifyEmailAddress(email, fromDomain, fromEmail string, dnsRecords dns.DNS) (SMPTValidation, error) {
	results := SMPTValidation{}

	if len(dnsRecords.MX) == 0 {
		results.Description = "No MX records for domain"
		if dnsRecords.SPF != "" {
			results.Description += fmt.Sprintf(". SPF record: %s", dnsRecords.SPF)
		}
		if len(dnsRecords.Errors) > 0 {
			results.Description += fmt.Sprintf(". Errors: %s", strings.Join(dnsRecords.Errors, ", "))
		}
		return results, nil
	}

	var conn net.Conn
	var client *bufio.Reader
	var err error
	var connected bool

	for i := 0; i < len(dnsRecords.MX); i++ {
		conn, client, err = connectToSMTP(dnsRecords.MX[i])
		if err != nil {
			continue
		}
		err = readSMTPgreeting(client)
		if err == nil {
			connected = true
			break
		}
	}

	if !connected {
		return results, errors.Wrap(err, "Failed to connect to any SMTP server")
	}

	defer conn.Close()

	if err = sendHELO(conn, client, fromDomain); err != nil {
		return results, errors.Wrap(err, "Failed to send HELO command")
	}

	if err = sendMAILFROM(conn, client, fromEmail); err != nil {
		return results, errors.Wrap(err, "Failed to send MAIL FROM command")
	}

	results, err = sendRCPTTO(conn, client, email)
	if err != nil {
		return results, errors.Wrap(err, "Failed to send RCPT TO command")
	}

	return results, nil
}

func connectToSMTP(mxServer string) (net.Conn, *bufio.Reader, error) {
	conn, err := net.DialTimeout("tcp", mxServer+":25", 10*time.Second)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect to SMTP server")
	}

	client := bufio.NewReader(conn)
	return conn, client, nil
}

func readSMTPgreeting(smtpClient *bufio.Reader) error {
	for {
		line, err := smtpClient.ReadString('\n')
		if err != nil {
			return errors.Wrap(err, "Failed to read SMTP server greeting")
		}

		// Trim the line to remove whitespace and newline characters
		line = strings.TrimSpace(line)

		// Check if this is the last line of the greeting
		if strings.HasPrefix(line, "220 ") {
			// This is the final greeting line
			return nil
		} else if !strings.HasPrefix(line, "220-") {
			// If the line doesn't start with 220- or 220, it's an unexpected response
			return fmt.Errorf("Unexpected SMTP server greeting: %s", line)
		}

		// If it's a continuation line (starts with 220-), continue reading
	}
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

func sendHELO(conn net.Conn, smtpClient *bufio.Reader, fromDomain string) error {
	helo := fmt.Sprintf("HELO %s", fromDomain)
	resp, err := sendSMTPcommand(conn, smtpClient, helo)
	if err != nil || !strings.HasPrefix(resp, "250") {
		return errors.New("HELO failed")
	}
	return nil
}

func sendMAILFROM(conn net.Conn, smtpClient *bufio.Reader, fromEmail string) error {
	mailfrom := fmt.Sprintf("MAIL FROM:<%s>", fromEmail)
	resp, err := sendSMTPcommand(conn, smtpClient, mailfrom)
	if err != nil || !strings.HasPrefix(resp, "250") {
		return errors.New("MAIL FROM failed")
	}
	return nil
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
