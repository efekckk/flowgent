// Package emailsmtp implements "email.smtp_send" — SMTP send with PLAIN auth
// via Go stdlib net/smtp. The credential payload carries host/port/username/
// password/from; the workflow node supplies to/subject/body. The Sender
// interface lets tests fake the network without standing up an SMTP server.
package emailsmtp

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct {
	sender Sender
}

func New() *Executor { return &Executor{sender: stdSender{}} }

func newWithSender(s Sender) *Executor { return &Executor{sender: s} }

func (e *Executor) Execute(_ context.Context, input map[string]any) (registry.ExecuteResult, error) {
	to, _ := input["to"].(string)
	if to == "" {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: missing \"to\"")
	}
	subject, _ := input["subject"].(string)
	if subject == "" {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: missing \"subject\"")
	}
	body, _ := input["body"].(string)
	if body == "" {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: missing \"body\"")
	}

	cred, ok := input["__credential"].(map[string]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: missing credential")
	}
	host, _ := cred["host"].(string)
	port, _ := cred["port"].(string)
	username, _ := cred["username"].(string)
	password, _ := cred["password"].(string)
	from, _ := cred["from"].(string)
	if host == "" || port == "" || username == "" || password == "" || from == "" {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: credential must include host, port, username, password, from")
	}

	recipients := splitRecipients(to)
	if len(recipients) == 0 {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: \"to\" contained no addresses")
	}

	auth := smtp.PlainAuth("", username, password, host)
	msg := buildMessage(from, recipients, subject, body)

	if err := e.sender.SendMail(host+":"+port, auth, from, recipients, msg); err != nil {
		// Do not wrap with %w — net/smtp errors generally don't carry the
		// password, but staying safe by returning a sanitized message.
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: send failed: %s", sanitizeErr(err, password))
	}

	return registry.ExecuteResult{
		Output: map[string]any{"sent": true, "to": to},
		Port:   "main",
	}, nil
}

func splitRecipients(to string) []string {
	parts := strings.Split(to, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// buildMessage returns RFC 5322 lines: From/To/Subject/Content-Type headers,
// a blank line, then the body. \r\n line endings as SMTP requires.
func buildMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// sanitizeErr redacts the password from an SMTP error message in the unlikely
// event the underlying library ever surfaces it. Defensive only.
func sanitizeErr(err error, password string) string {
	s := err.Error()
	if password != "" {
		s = strings.ReplaceAll(s, password, "<redacted>")
	}
	return s
}
