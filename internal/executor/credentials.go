package executor

import "context"

// CredentialResolver is the contract the engine uses to translate a workflow
// node's `credential` string reference into a decrypted secret payload.
// Implementations live in the api/main wiring layer where the storage
// repository and crypto key are available.
type CredentialResolver interface {
	Resolve(ctx context.Context, credentialRef string) (map[string]any, error)
}

// WithCredentialResolver attaches a CredentialResolver to the engine.
func WithCredentialResolver(r CredentialResolver) Option {
	return func(e *Engine) { e.credResolver = r }
}
