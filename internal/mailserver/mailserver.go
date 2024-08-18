package mailserver

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

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

type ProxySetup struct {
	Enable   bool
	Address  string
	Username string
	Password string
}

func VerifyEmailAddress(email, fromDomain, fromEmail string, dnsRecords dns.DNS) (bool, SMPTValidation, error) {
	results := SMPTValidation{}
	var isVerified bool

	if len(dnsRecords.MX) == 0 {
		results.Description = "No MX records for domain"
		return false, results, nil
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
		err := readSMTPgreeting(client)
		if err == nil {
			connected = true
			break
		}
	}

	if !connected {
		return false, results, fmt.Errorf("failed to connect to any MX server: %w", err)
	}

	defer conn.Close()

	if err := sendHELO(conn, client, fromDomain); err != nil {
		return false, results, err
	}

	if err := sendMAILFROM(conn, client, fromEmail); err != nil {
		return false, results, err
	}

	isVerified, results, err = sendRCPTTO(conn, client, email)
	if err != nil {
		return false, SMPTValidation{}, err
	}

	return isVerified, results, nil
}

func connectToSMTP(mxServer string) (net.Conn, *bufio.Reader, error) {
	conn, err := net.DialTimeout("tcp", mxServer+":25", 10*time.Second)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to connect to SMTP server: %w", err)
	}

	client := bufio.NewReader(conn)
	return conn, client, nil
}

func readSMTPgreeting(smtpClient *bufio.Reader) error {
	for {
		line, err := smtpClient.ReadString('\n')
		if err != nil {
			return fmt.Errorf("Failed to read SMTP server greeting: %w", err)
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
		return "", fmt.Errorf("failed to send SMTP command %s: %w", cmd, err)
	}
	resp, err := smtpClient.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read response for SMTP command %s: %w", cmd, err)
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

func sendRCPTTO(conn net.Conn, smtpClient *bufio.Reader, emailToValidate string) (isValid bool, results SMPTValidation, err error) {
	rcpt := fmt.Sprintf("RCPT TO:<%s>", emailToValidate)
	resp, err := sendSMTPcommand(conn, smtpClient, rcpt)
	if err != nil {
		return false, results, fmt.Errorf("RCPT TO command failed: %w", err)
	}

	results.SmtpResponse = resp
	results.ResponseCode, results.ErrorCode, results.Description = ParseSmtpResponse(resp)

	switch results.ResponseCode {
	case "250":
		results.CanConnectSmtp = true
		isValid = true
	case "251":
		results.CanConnectSmtp = true
		isValid = true
	case "552":
		results.InboxFull = true
		results.CanConnectSmtp = true
		isValid = false
	default:
		results.CanConnectSmtp = true
		isValid = false
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
