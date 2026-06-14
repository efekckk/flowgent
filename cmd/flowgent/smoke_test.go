//go:build smoke

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/api"
	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func TestM1Smoke_signupLoginMeLogout(t *testing.T) {
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		t.Skip("docker unavailable")
	}
	if err := storagetest.Start(); err != nil {
		t.Skipf("dockertest: %v", err)
	}
	defer storagetest.Stop()
	dsn := storagetest.Fresh(t)

	pg, err := storage.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pg.Close()

	srv := api.NewServer(api.Deps{
		Users:        storage.NewUserRepo(pg.Pool),
		Workspaces:   storage.NewWorkspaceRepo(pg.Pool),
		Sessions:     storage.NewSessionRepo(pg.Pool),
		Throttle:     auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain: "",
		CookieSecure: false,
	})

	ts := http.Server{Addr: ":0", Handler: srv}
	ln, err := startListener(t)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go ts.Serve(ln)
	defer ts.Close()

	base := "http://" + ln.Addr().String()
	jar, _ := cookiejar.New(nil)
	cli := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	signup := mustJSON(map[string]string{"email": "smoke@example.com", "password": "supersecret"})
	rr, err := cli.Post(base+"/v1/auth/signup", "application/json", bytes.NewReader(signup))
	if err != nil || rr.StatusCode != http.StatusCreated {
		t.Fatalf("signup: %v %d", err, rr.StatusCode)
	}

	rr, err = cli.Get(base + "/v1/me")
	if err != nil || rr.StatusCode != http.StatusOK {
		t.Fatalf("me: %v %d", err, rr.StatusCode)
	}

	rr, err = cli.Post(base+"/v1/auth/logout", "application/json", nil)
	if err != nil || rr.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %v %d", err, rr.StatusCode)
	}

	rr, err = cli.Get(base + "/v1/me")
	if err != nil || rr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout /me: %v %d", err, rr.StatusCode)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
