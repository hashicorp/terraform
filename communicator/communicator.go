package communicator

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/communicator/ssh"
	"github.com/hashicorp/terraform/communicator/winrm"
	"github.com/hashicorp/terraform/terraform"
)

// Communicator is an interface that must be implemented by all communicators
// used for any of the provisioners
type Communicator interface {
	// Connect is used to setup the connection
	Connect(terraform.UIOutput) error

	// Disconnect is used to terminate the connection
	Disconnect() error

	// Timeout returns the configured connection timeout
	Timeout() time.Duration

	// ScriptPath returns the configured script path
	ScriptPath() string

	// Start executes a remote command in a new session
	Start(*remote.Cmd) error

	// Upload is used to upload a single file
	Upload(string, io.Reader) error

	// UploadScript is used to upload a file as a executable script
	UploadScript(string, io.Reader) error

	// UploadDir is used to upload a directory
	UploadDir(string, string) error
}

// New returns a configured Communicator or an error if the connection type is not supported
func New(s *terraform.InstanceState) (Communicator, error) {
	connType := s.Ephemeral.ConnInfo["type"]
	switch connType {
	case "ssh", "": // The default connection type is ssh, so if connType is empty use ssh
		return ssh.New(s)
	case "winrm":
		return winrm.New(s)
	default:
		return nil, fmt.Errorf("connection type '%s' not supported", connType)
	}
}

// maxBackoffDealy is the maximum delay between retry attempts
var maxBackoffDelay = 10 * time.Second
var initialBackoffDelay = time.Second

type Fatal interface {
	FatalError() error
}

func Retry(ctx context.Context, f func() error) error {
	// container for atomic error value
	type errWrap struct {
		E error
	}

	// Try the function in a goroutine
	var errVal atomic.Value
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		delay := time.Duration(0)
		for {
			// If our context ended, we want to exit right away.
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			// Try the function call
			err := f()

			// return if we have no error, or a FatalError
			done := false
			switch e := err.(type) {
			case nil:
				done = true
			case Fatal:
				err = e.FatalError()
				done = true
			}

			errVal.Store(&errWrap{err})

			if done {
				return
			}

			log.Printf("[WARN] retryable error: %v", err)

			delay *= 2

			if delay == 0 {
				delay = initialBackoffDelay
			}

			if delay > maxBackoffDelay {
				delay = maxBackoffDelay
			}

			log.Printf("[INFO] sleeping for %s", delay)
		}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
	case <-doneCh:
	}

	var lastErr error
	// Check if we got an error executing
	if ev, ok := errVal.Load().(errWrap); ok {
		lastErr = ev.E
	}

	// Check if we have a context error to check if we're interrupted or timeout
	switch ctx.Err() {
	case context.Canceled:
		return fmt.Errorf("interrupted - last error: %v", lastErr)
	case context.DeadlineExceeded:
		return fmt.Errorf("timeout - last error: %v", lastErr)
	}

	if lastErr != nil {
		return lastErr
	}
	return nil
}
