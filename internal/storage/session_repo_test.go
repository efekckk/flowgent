package storage_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func sha(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func TestSessionRepo_InsertAndFind(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	sess := storage.NewSessionRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "s@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)

	hash := sha("plaintext-token")
	exp := time.Now().Add(24 * time.Hour)
	s := storage.Session{
		TokenHash: hash, UserID: u.ID, ExpiresAt: exp, IP: "127.0.0.1", UserAgent: "test",
	}
	if err := sess.Insert(ctx, s); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := sess.FindByTokenHash(ctx, hash)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.UserID != u.ID {
		t.Errorf("want user %s, got %s", u.ID, got.UserID)
	}
}

func TestSessionRepo_FindByTokenHash_Expired(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	sess := storage.NewSessionRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "exp@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)

	hash := sha("expired-token")
	_ = sess.Insert(ctx, storage.Session{
		TokenHash: hash, UserID: u.ID, ExpiresAt: time.Now().Add(-time.Minute),
	})
	_, err := sess.FindByTokenHash(ctx, hash)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired session, got %v", err)
	}
}

func TestSessionRepo_Delete(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	sess := storage.NewSessionRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "del@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	hash := sha("kill-me")
	_ = sess.Insert(ctx, storage.Session{TokenHash: hash, UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)})

	if err := sess.Delete(ctx, hash); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := sess.FindByTokenHash(ctx, hash); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected gone, got %v", err)
	}
}
