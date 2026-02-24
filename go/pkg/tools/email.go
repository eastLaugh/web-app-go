package tools

import (
	"encoding/base64"
	"net/smtp"
	"os"
	"strings"
)

func SendMail(subject string, body string, to ...string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")

	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	const lineLen = 76
	var folded strings.Builder
	for i := 0; i < len(encoded); i += lineLen {
		end := i + lineLen
		if end > len(encoded) {
			end = len(encoded)
		}
		folded.WriteString(encoded[i:end])
		folded.WriteString("\r\n")
	}

	header := "To: " + strings.Join(to, ",") + "\r\n" +
		"From: " + user + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n"
	msg := []byte(header + folded.String())
	return smtp.SendMail(host+":"+port, smtp.PlainAuth("", user, pass, host), user, to, msg)
}
