package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform/terraform"
)

// State is the collection of all state interfaces.
type State interface {
	StateReader
	StateWriter
	StateRefresher
	StatePersister
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

// LockInfo stores metadata for locks taken.
type LockInfo struct {
	ID        string    // unique ID
	Path      string    // Path to the state file
	Created   time.Time // The time the lock was taken
	Version   string    // Terraform version
	Operation string    // Terraform operation
	Who       string    // user@hostname when available
	Info      string    // Extra info field
}

// Err returns the lock info formatted in an error
func (l *LockInfo) Err() error {
	return fmt.Errorf("state locked. path:%q, created:%s, info:%q",
		l.Path, l.Created, l.Info)
}

func (l *LockInfo) String() string {
	js, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}
	return string(js)
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
		out = append(out, e.Info.Err().Error())
	}
	return strings.Join(out, "\n")
}
