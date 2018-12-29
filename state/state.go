package state

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"time"

	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/version"
)

var rngSource *rand.Rand

func init() {
	rngSource = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// State is a deprecated alias for statemgr.Full
type State = statemgr.Full

// StateReader is a deprecated alias for statemgr.Reader
type StateReader = statemgr.Reader

// StateWriter is a deprecated alias for statemgr.Writer
type StateWriter = statemgr.Writer

// StateRefresher is a deprecated alias for statemgr.Refresher
type StateRefresher = statemgr.Refresher

// StatePersister is a deprecated alias for statemgr.Persister
type StatePersister = statemgr.Persister

// Locker is a deprecated alias for statemgr.Locker
type Locker = statemgr.Locker

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
		Version: version.Version,
		Created: time.Now().UTC(),
	}
	return info
}

// LockInfo is a deprecated lias for statemgr.LockInfo
type LockInfo = statemgr.LockInfo

// LockError is a deprecated alias for statemgr.LockError
type LockError = statemgr.LockError
