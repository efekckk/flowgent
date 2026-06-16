// Package emailsmtp implements "email.smtp_send" — SMTP send with PLAIN auth
// via Go stdlib net/smtp. The credential payload carries host/port/username/
// password/from; the workflow node supplies to/subject/body. The Sender
// interface lets tests fake the network without standing up an SMTP server.
package emailsmtp

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/efekckk/flowgent/internal/executor"
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

	portN, err := strconv.Atoi(port)
	if err != nil || portN < 1 || portN > 65535 {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: credential \"port\" must be 1..65535: %w", executor.ErrValidation)
	}

	recipients := splitRecipients(to)
	if len(recipients) == 0 {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: \"to\" contained no addresses")
	}

	if containsHeaderControlChar(subject) || containsHeaderControlChar(from) {
		return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: header field contains control characters: %w", executor.ErrValidation)
	}
	for _, addr := range recipients {
		if containsHeaderControlChar(addr) {
			return registry.ExecuteResult{}, fmt.Errorf("email.smtp_send: recipient address contains control characters: %w", executor.ErrValidation)
		}
	}

	auth := smtp.PlainAuth("", username, password, host)
	msg := buildMessage(from, recipients, subject, body)

	if err := e.sender.SendMail(fmt.Sprintf("%s:%d", host, portN), auth, from, recipients, msg); err != nil {
		return registry.ExecuteResult{}, classifySendErr(err, password)
	}

	return registry.ExecuteResult{
		Output: map[string]any{
			"sent":       true,
			"to":         to,
			"recipients": recipients,
		},
		Port: "main",
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

// classifySendErr maps SMTP send failures to the engine's retry sentinels.
// *textproto.Error carries the SMTP reply code; we use those classes when
// available. Network-level errors (dial, TLS, timeout) carry no code — they
// are treated as transient since most are recoverable on the next attempt.
// The error message is sanitized so the credential password is never echoed
// even if the underlying library ever surfaces it.
func classifySendErr(err error, password string) error {
	msg := sanitizeErr(err, password)
	var te *textproto.Error
	if errors.As(err, &te) {
		switch {
		case te.Code == 421, te.Code == 450, te.Code == 451, te.Code == 452:
			return fmt.Errorf("email.smtp_send: send failed (%d): %s: %w", te.Code, msg, executor.ErrTransient5xx)
		case te.Code == 530, te.Code == 535:
			return fmt.Errorf("email.smtp_send: auth failed (%d): %s: %w", te.Code, msg, executor.ErrAuthFailed)
		case te.Code >= 400 && te.Code < 500:
			return fmt.Errorf("email.smtp_send: send failed (%d): %s: %w", te.Code, msg, executor.ErrTransient5xx)
		case te.Code >= 500:
			return fmt.Errorf("email.smtp_send: send rejected (%d): %s: %w", te.Code, msg, executor.ErrValidation)
		}
	}
	// No textproto code — assume network-level transient failure.
	return fmt.Errorf("email.smtp_send: transport error: %s: %w", msg, executor.ErrTransient5xx)
}

// containsHeaderControlChar reports whether s contains a CR, LF, or NUL byte.
// These bytes must never reach the SMTP wire as part of header values; they
// allow an attacker who controls upstream node output to inject additional
// headers (CWE-93). The check is byte-level on purpose — a single byte is
// enough for the wire interpreter to treat the rest of the line as a new
// header or body.
func containsHeaderControlChar(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '\r' || c == '\n' || c == 0 {
			return true
		}
	}
	return false
}
