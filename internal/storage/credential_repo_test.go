package storage_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func setupCredentialContext(t *testing.T) (*storage.CredentialRepo, storage.Workspace) {
	t.Helper()
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	creds := storage.NewCredentialRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "cred@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)
	return creds, ws
}

func TestCredentialRepo_InsertAndGet(t *testing.T) {
	creds, ws := setupCredentialContext(t)
	ctx := context.Background()

	cred := storage.Credential{
		ID:          idgen.NewCredential(),
		WorkspaceID: ws.ID,
		Name:        "openai_default",
		Type:        "openai",
		Encrypted:   []byte{0xde, 0xad, 0xbe, 0xef},
		Meta:        json.RawMessage(`{"base_url":"https://api.openai.com/v1"}`),
	}
	if err := creds.Insert(ctx, cred); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := creds.Get(ctx, cred.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "openai_default" || got.Type != "openai" {
		t.Errorf("got %+v", got)
	}
	if len(got.Encrypted) != 4 || got.Encrypted[0] != 0xde {
		t.Errorf("encrypted byte round-trip lost: %x", got.Encrypted)
	}
}

func TestCredentialRepo_DuplicateNameWithinWorkspaceConflict(t *testing.T) {
	creds, ws := setupCredentialContext(t)
	ctx := context.Background()

	c1 := storage.Credential{
		ID: idgen.NewCredential(), WorkspaceID: ws.ID,
		Name: "dup", Type: "openai", Encrypted: []byte{1},
	}
	_ = creds.Insert(ctx, c1)

	c2 := storage.Credential{
		ID: idgen.NewCredential(), WorkspaceID: ws.ID,
		Name: "dup", Type: "anthropic", Encrypted: []byte{2},
	}
	err := creds.Insert(ctx, c2)
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCredentialRepo_ListByWorkspaceFiltersScope(t *testing.T) {
	creds, ws := setupCredentialContext(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_ = creds.Insert(ctx, storage.Credential{
			ID: idgen.NewCredential(), WorkspaceID: ws.ID,
			Name: "k" + string(rune('a'+i)), Type: "openai", Encrypted: []byte{1},
		})
	}
	rows, err := creds.ListByWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("count: %d", len(rows))
	}
}

func TestCredentialRepo_Delete(t *testing.T) {
	creds, ws := setupCredentialContext(t)
	ctx := context.Background()
	cred := storage.Credential{
		ID: idgen.NewCredential(), WorkspaceID: ws.ID,
		Name: "kill", Type: "openai", Encrypted: []byte{1},
	}
	_ = creds.Insert(ctx, cred)
	if err := creds.Delete(ctx, cred.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := creds.Get(ctx, cred.ID)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
