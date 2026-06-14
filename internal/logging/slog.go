// Package logging wraps log/slog with a JSON handler and a key-based redactor.
// Any attribute key matching the redactKeys set is replaced with "[REDACTED]"
// before serialisation, so secrets cannot accidentally land in logs. Attributes
// nested under a sensitive group name are suppressed entirely.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

var redactKeys = map[string]struct{}{
	"password":      {},
	"password_hash": {},
	"session_token": {},
	"token":         {},
	"api_key":       {},
	"auth_key":      {},
	"authorization": {},
	"cred_payload":  {},
	"webhook_token": {},
}

func NewLogger(w io.Writer, level string) *slog.Logger {
	lvl := parseLevel(level)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Suppress every leaf whose ancestor group name is sensitive.
			for _, g := range groups {
				if _, hit := redactKeys[strings.ToLower(g)]; hit {
					return slog.Attr{}
				}
			}
			if _, hit := redactKeys[strings.ToLower(a.Key)]; hit {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	})
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
