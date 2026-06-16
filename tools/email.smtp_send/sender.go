// Package emailsmtp implements the SMTP send abstraction. The Sender
// interface mirrors net/smtp.SendMail so production code uses the stdlib
// directly and tests can inject a fake without spinning up a real SMTP
// listener.
package emailsmtp

import "net/smtp"

// Sender abstracts the SMTP send so executor tests can substitute a fake.
type Sender interface {
	SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

type stdSender struct{}

func (stdSender) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, a, from, to, msg)
}
