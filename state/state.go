package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"strings"
	"text/template"
	"time"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/terraform"
)

var rngSource *rand.Rand

func init() {
	rngSource = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// State is the collection of all state interfaces.
type State interface {
	StateReader
	StateWriter
	StateRefresher
	StatePersister
	Locker
}

// StateReader is the interface for things that can return a state. Retrieving
// the state here must not error. Loading the state fresh (an operation that
// can likely error) should be implemented by RefreshState. If a state hasn't
// been loaded yet, it is okay for State to return nil.
type StateReader interface {
	State() *terraform.State
}

// StateWriter is the interface that must be implemented by something that
// can write a state. Writing the state can be cached or in-memory, as
// full persistence should be implemented by StatePersister.
type StateWriter interface {
	WriteState(*terraform.State) error
}

// StateRefresher is the interface that is implemented by something that
// can load a state. This might be refreshing it from a remote location or
// it might simply be reloading it from disk.
type StateRefresher interface {
	RefreshState() error
}

// StatePersister is implemented to truly persist a state. Whereas StateWriter
// is allowed to perhaps be caching in memory, PersistState must write the
// state to some durable storage.
type StatePersister interface {
	PersistState() error
}

// Locker is implemented to lock state during command execution.
// The info parameter can be recorded with the lock, but the
// implementation should not depend in its value. The string returned by Lock
// is an ID corresponding to the lock acquired, and must be passed to Unlock to
// ensure that the correct lock is being released.
//
// Lock and Unlock may return an error value of type LockError which in turn
// can contain the LockInfo of a conflicting lock.
type Locker interface {
	Lock(info *LockInfo) (string, error)
	Unlock(id string) error
}

// test hook to verify that LockWithContext has attempted a lock
var postLockHook func()

// Lock the state, using the provided context for timeout and cancellation.
// This backs off slightly to an upper limit.
func LockWithContext(ctx context.Context, s State, info *LockInfo) (string, error) {
	delay := time.Second
	maxDelay := 16 * time.Second
	for {
		id, err := s.Lock(info)
		if err == nil {
			return id, nil
		}

		le, ok := err.(*LockError)
		if !ok {
			// not a lock error, so we can't retry
			return "", err
		}

		if le == nil || le.Info == nil || le.Info.ID == "" {
			// If we dont' have a complete LockError, there's something wrong with the lock
			return "", err
		}

		if postLockHook != nil {
			postLockHook()
		}

		// there's an existing lock, wait and try again
		select {
		case <-ctx.Done():
			// return the last lock error with the info
			return "", err
		case <-time.After(delay):
			if delay < maxDelay {
				delay *= 2
			}
		}
	}
}

// Generate a LockInfo structure, populating the required fields.
func NewLockInfo() *LockInfo {
	// this doesn't need to be cryptographically secure, just unique.
	// Using math/rand alleviates the need to check handle the read error.
	// Use a uuid format to match other IDs used throughout Terraform.
	buf := make([]byte, 16)
	rngSource.Read(buf)

	id, err := uuid.FormatUUID(buf)
	if err != nil {
		// this of course shouldn't happen
		panic(err)
	}

	// don't error out on user and hostname, as we don't require them
	userName := ""
	if userInfo, err := user.Current(); err == nil {
		userName = userInfo.Username
	}
	host, _ := os.Hostname()

	info := &LockInfo{
		ID:      id,
		Who:     fmt.Sprintf("%s@%s", userName, host),
		Version: terraform.Version,
		Created: time.Now().UTC(),
	}
	return info
}

// LockInfo stores lock metadata.
//
// Only Operation and Info are required to be set by the caller of Lock.
type LockInfo struct {
	// Unique ID for the lock. NewLockInfo provides a random ID, but this may
	// be overridden by the lock implementation. The final value if ID will be
	// returned by the call to Lock.
	ID string

	// Terraform operation, provided by the caller.
	Operation string
	// Extra information to store with the lock, provided by the caller.
	Info string

	// user@hostname when available
	Who string
	// Terraform version
	Version string
	// Time that the lock was taken.
	Created time.Time

	// Path to the state file when applicable. Set by the Lock implementation.
	Path string
}

// Err returns the lock info formatted in an error
func (l *LockInfo) Err() error {
	return errors.New(l.String())
}

// Marshal returns a string json representation of the LockInfo
func (l *LockInfo) Marshal() []byte {
	js, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}
	return js
}

// String return a multi-line string representation of LockInfo
func (l *LockInfo) String() string {
	tmpl := `Lock Info:
  ID:        {{.ID}}
  Path:      {{.Path}}
  Operation: {{.Operation}}
  Who:       {{.Who}}
  Version:   {{.Version}}
  Created:   {{.Created}}
  Info:      {{.Info}}
`

	t := template.Must(template.New("LockInfo").Parse(tmpl))
	var out bytes.Buffer
	if err := t.Execute(&out, l); err != nil {
		panic(err)
	}
	return out.String()
}

type LockError struct {
	Info *LockInfo
	Err  error
}

func (e *LockError) Error() string {
	var out []string
	if e.Err != nil {
		out = append(out, e.Err.Error())
	}

	if e.Info != nil {
		out = append(out, e.Info.String())
	}
	return strings.Join(out, "\n")
}
