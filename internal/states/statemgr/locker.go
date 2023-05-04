// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statemgr

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
	"github.com/hashicorp/terraform/version"
)

var rngSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// Locker is the interface for state managers that are able to manage
// mutual-exclusion locks for state.
//
// Implementing Locker alongside Persistent relaxes some of the usual
// implementation constraints for implementations of Refresher and Persister,
// under the assumption that the locking mechanism effectively prevents
// multiple Terraform processes from reading and writing state concurrently.
// In particular, a type that implements both Locker and Persistent is only
// required to that the Persistent implementation is concurrency-safe within
// a single Terraform process.
//
// A Locker implementation must ensure that another processes with a
// similarly-configured state manager cannot successfully obtain a lock while
// the current process is holding it, or vice-versa, assuming that both
// processes agree on the locking mechanism.
//
// A Locker is not required to prevent non-cooperating processes from
// concurrently modifying the state, but is free to do so as an extra
// protection. If a mandatory locking mechanism of this sort is implemented,
// the state manager must ensure that RefreshState and PersistState calls
// can succeed if made through the same manager instance that is holding the
// lock, such has by retaining some sort of lock token that the Persistent
// methods can then use.
type Locker interface {
	// Lock attempts to obtain a lock, using the given lock information.
	//
	// The result is an opaque id that can be passed to Unlock to release
	// the lock, or an error if the lock cannot be acquired. Lock returns
	// an instance of LockError immediately if the lock is already held,
	// and the helper function LockWithContext uses this to automatically
	// retry lock acquisition periodically until a timeout is reached.
	Lock(info *LockInfo) (string, error)

	// Unlock releases a lock previously acquired by Lock.
	//
	// If the lock cannot be released -- for example, if it was stolen by
	// another user with some sort of administrative override privilege --
	// then an error is returned explaining the situation in a way that
	// is suitable for returning to an end-user.
	Unlock(id string) error
}

// test hook to verify that LockWithContext has attempted a lock
var postLockHook func()

// LockWithContext locks the given state manager using the provided context
// for both timeout and cancellation.
//
// This method has a built-in retry/backoff behavior up to the context's
// timeout.
func LockWithContext(ctx context.Context, s Locker, info *LockInfo) (string, error) {
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
			// If we don't have a complete LockError then there's something
			// wrong with the lock.
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

// LockInfo stores lock metadata.
//
// Only Operation and Info are required to be set by the caller of Lock.
// Most callers should use NewLockInfo to create a LockInfo value with many
// of the fields populated with suitable default values.
type LockInfo struct {
	// Unique ID for the lock. NewLockInfo provides a random ID, but this may
	// be overridden by the lock implementation. The final value of ID will be
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

// NewLockInfo creates a LockInfo object and populates many of its fields
// with suitable default values.
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
		Version: version.Version,
		Created: time.Now().UTC(),
	}
	return info
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

// LockError is a specialization of type error that is returned by Locker.Lock
// to indicate that the lock is already held by another process and that
// retrying may be productive to take the lock once the other process releases
// it.
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
