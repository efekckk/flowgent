package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func openFresh(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := storagetest.Fresh(t)
	ctx := context.Background()
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestUserRepo_InsertAndFindByEmail(t *testing.T) {
	pool := openFresh(t)
	repo := storage.NewUserRepo(pool)
	ctx := context.Background()

	u := storage.User{
		ID:           idgen.NewUser(),
		Email:        "alice@example.com",
		PasswordHash: "argon2id$placeholder",
	}
	if err := repo.Insert(ctx, u); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := repo.FindByEmail(ctx, "ALICE@example.com") // citext case-insensitive
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != u.ID || got.Email != "alice@example.com" {
		t.Errorf("got %+v, want id=%s email=%s", got, u.ID, u.Email)
	}
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	pool := openFresh(t)
	repo := storage.NewUserRepo(pool)
	_, err := repo.FindByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepo_Insert_DuplicateEmail(t *testing.T) {
	pool := openFresh(t)
	repo := storage.NewUserRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "dup@example.com", PasswordHash: "h"}
	if err := repo.Insert(ctx, u); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	u.ID = idgen.NewUser()
	err := repo.Insert(ctx, u)
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate email, got %v", err)
	}
}

func TestUserRepo_FindByID(t *testing.T) {
	pool := openFresh(t)
	repo := storage.NewUserRepo(pool)
	ctx := context.Background()

	u := storage.User{
		ID:           idgen.NewUser(),
		Email:        "byid@example.com",
		PasswordHash: "argon2id$placeholder",
	}
	if err := repo.Insert(ctx, u); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != u.ID || got.Email != u.Email {
		t.Errorf("got %+v, want id=%s email=%s", got, u.ID, u.Email)
	}
}

func TestUserRepo_FindByID_NotFound(t *testing.T) {
	pool := openFresh(t)
	repo := storage.NewUserRepo(pool)
	_, err := repo.FindByID(context.Background(), "usr_doesnotexist")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
