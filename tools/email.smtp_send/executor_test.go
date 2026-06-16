package emailsmtp

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"
)

type fakeSender struct {
	addr string
	from string
	to   []string
	msg  []byte
	err  error
}

func (f *fakeSender) SendMail(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
	f.addr = addr
	f.from = from
	f.to = to
	f.msg = msg
	return f.err
}

func smtpCred() map[string]any {
	return map[string]any{
		"host":     "smtp.example.com",
		"port":     "587",
		"username": "user",
		"password": "pw",
		"from":     "bot@example.com",
		"__type":   "smtp",
	}
}

func TestExecute_buildsAndSendsMessage(t *testing.T) {
	fs := &fakeSender{}
	e := newWithSender(fs)
	res, err := e.Execute(context.Background(), map[string]any{
		"to":           "alice@example.com",
		"subject":      "Hello",
		"body":         "World",
		"__credential": smtpCred(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if fs.addr != "smtp.example.com:587" {
		t.Errorf("addr: %s", fs.addr)
	}
	if fs.from != "bot@example.com" || len(fs.to) != 1 || fs.to[0] != "alice@example.com" {
		t.Errorf("from/to: %s / %+v", fs.from, fs.to)
	}
	msg := string(fs.msg)
	if !strings.Contains(msg, "Subject: Hello\r\n") {
		t.Errorf("subject header missing: %s", msg)
	}
	if !strings.Contains(msg, "To: alice@example.com\r\n") {
		t.Errorf("to header missing: %s", msg)
	}
	if !strings.Contains(msg, "From: bot@example.com\r\n") {
		t.Errorf("from header missing: %s", msg)
	}
	if !strings.HasSuffix(msg, "\r\n\r\nWorld") {
		t.Errorf("body separator missing: %s", msg)
	}
	if res.Output["sent"] != true {
		t.Errorf("output sent: %+v", res.Output)
	}
}

func TestExecute_multipleRecipientsSplitsCommaList(t *testing.T) {
	fs := &fakeSender{}
	e := newWithSender(fs)
	_, err := e.Execute(context.Background(), map[string]any{
		"to":           "a@x.com, b@x.com,c@x.com",
		"subject":      "x",
		"body":         "y",
		"__credential": smtpCred(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(fs.to) != 3 {
		t.Fatalf("recipients: %+v", fs.to)
	}
	if fs.to[0] != "a@x.com" || fs.to[1] != "b@x.com" || fs.to[2] != "c@x.com" {
		t.Errorf("recipients order/trim: %+v", fs.to)
	}
}

func TestExecute_missingToIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"subject":      "x",
		"body":         "y",
		"__credential": smtpCred(),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingSubjectOrBodyIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"to":           "a@x.com",
		"body":         "y",
		"__credential": smtpCred(),
	})
	if err == nil {
		t.Fatalf("expected error when subject missing")
	}
	_, err = e.Execute(context.Background(), map[string]any{
		"to":           "a@x.com",
		"subject":      "s",
		"__credential": smtpCred(),
	})
	if err == nil {
		t.Fatalf("expected error when body missing")
	}
}

func TestExecute_missingCredentialFieldsIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"to":           "x@y",
		"subject":      "s",
		"body":         "b",
		"__credential": map[string]any{"host": "h", "__type": "smtp"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingCredentialIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"to":      "x@y",
		"subject": "s",
		"body":    "b",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_senderErrorPropagates(t *testing.T) {
	fs := &fakeSender{err: errors.New("connection refused")}
	e := newWithSender(fs)
	_, err := e.Execute(context.Background(), map[string]any{
		"to":           "a@x.com",
		"subject":      "s",
		"body":         "b",
		"__credential": smtpCred(),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), "pw") {
		t.Errorf("password leaked into error: %v", err)
	}
}
