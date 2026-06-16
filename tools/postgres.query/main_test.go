package postgresquery

import (
	"log"
	"os"
	"testing"

	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func TestMain(m *testing.M) {
	if err := storagetest.Start(); err != nil {
		log.Printf("postgres.query: skipping (docker unavailable): %v", err)
		os.Exit(m.Run())
	}
	code := m.Run()
	storagetest.Stop()
	os.Exit(code)
}
