// Package idgen produces prefix-tagged ULIDs ("usr_01H...") used as primary
// keys across Flowgent. ULIDs sort lexicographically by creation time, which
// keeps cursor pagination simple. Prefixes make IDs self-describing at a
// glance, especially in logs.
package idgen

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

func newID(prefix string) string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return prefix + id.String()
}

func NewUser() string            { return newID("usr_") }
func NewWorkspace() string       { return newID("ws_") }
func NewSession() string         { return newID("sess_") }
func NewCredential() string      { return newID("cred_") }
func NewWorkflow() string        { return newID("wf_") }
func NewWorkflowVersion() string { return newID("wfv_") }
func NewRun() string             { return newID("run_") }
func NewNodeRun() string         { return newID("nrun_") }
func NewChatThread() string      { return newID("thr_") }
func NewChatMessage() string     { return newID("msg_") }
