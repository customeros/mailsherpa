package mailserver

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	"github.com/customeros/mailsherpa/internal/dns"
)

type SMPTValidation struct {
	CanConnectSmtp bool
	InboxFull      bool
	SMTPError      string
}

type ProxySetup struct {
	Enable   bool
	Address  string
	Username string
	Password string
}

func VerifyEmailAddress(email, fromDomain, fromEmail string, proxy ProxySetup) (bool, SMPTValidation, error) {
	results := SMPTValidation{}
	var isVerified bool

	mxServers, err := dns.GetMXRecordsForEmail(email)
	if err != nil {
		return false, results, err
	}

	if len(mxServers) == 0 {
		return false, results, fmt.Errorf("no MX records found for domain")
	}

	var conn net.Conn
	var client *bufio.Reader

	if proxy.Enable {
		fmt.Println("Enabling proxy...")
		conn, client, err = connectToSMTPviaProxy(mxServers[0], proxy.Address, proxy.Username, proxy.Password)
	} else {
		conn, client, err = connectToSMTP(mxServers[0])
	}
	if err != nil {
		return false, results, err
	}

	defer conn.Close()

	if err := readSMTPgreeting(client); err != nil {
		return false, results, err
	}

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

func connectToSMTPviaProxy(mxServer, proxyAddress, proxyUsername, proxyPassword string) (net.Conn, *bufio.Reader, error) {
	auth := &proxy.Auth{
		User:     proxyUsername,
		Password: proxyPassword,
	}

	dialer, err := proxy.SOCKS5("tcp", proxyAddress, auth, proxy.Direct)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to connect proxy dialer: %w", err)
	}

	conn, err := dialer.Dial("tcp", mxServer+":25")
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to connect to SMTP server via proxy: %w", err)
	}

	err = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("Failed to set connection deadline: %w", err)
	}

	client := bufio.NewReader(conn)
	return conn, client, nil
}

func readSMTPgreeting(smtpClient *bufio.Reader) error {
	_, err := smtpClient.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Failed to read SMTP server greeting: %w", err)
	}
	return nil
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

func sendRCPTTO(conn net.Conn, smtpClient *bufio.Reader, emailToValidate string) (bool, SMPTValidation, error) {
	results := SMPTValidation{}
	rcpt := fmt.Sprintf("RCPT TO:<%s>", emailToValidate)
	resp, err := sendSMTPcommand(conn, smtpClient, rcpt)
	if err != nil {
		return false, results, fmt.Errorf("RCPT TO command failed: %w", err)
	}

	respCode := strings.SplitN(resp, " ", 2)[0]

	switch respCode {
	case "250":
		results.CanConnectSmtp = true
		return true, results, nil
	case "251":
		results.CanConnectSmtp = true
		return true, results, nil
	case "450":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "451":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "452":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "503":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "550":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "551":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "552":
		results.InboxFull = true
		results.CanConnectSmtp = true
		return false, results, nil
	case "553":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	case "554":
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	default:
		results.CanConnectSmtp = true
		error := fmt.Sprintf("%s", resp)
		results.SMTPError = error
		return false, results, nil
	}
}
