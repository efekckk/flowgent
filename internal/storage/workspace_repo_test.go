package storage_test

import (
	"context"
	"testing"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func TestWorkspaceRepo_InsertAndFindByOwner(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "owner@example.com", PasswordHash: "h"}
	if err := users.Insert(ctx, u); err != nil {
		t.Fatalf("user insert: %v", err)
	}

	w := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "default"}
	if err := workspaces.Insert(ctx, w); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	got, err := workspaces.FindByOwner(ctx, u.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 1 || got[0].ID != w.ID {
		t.Errorf("expected 1 workspace %s, got %+v", w.ID, got)
	}
}

func TestWorkspaceRepo_CascadeOnUserDelete(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "cascade@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	w := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "default"}
	_ = workspaces.Insert(ctx, w)

	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, u.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	rows, err := workspaces.FindByOwner(ctx, u.ID)
	if err != nil {
		t.Fatalf("find after cascade: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected cascade delete, got %d", len(rows))
	}
}
