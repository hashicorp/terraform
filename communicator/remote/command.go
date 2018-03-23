package remote

import (
	"fmt"
	"io"
	"sync"
)

// Cmd represents a remote command being prepared or run.
type Cmd struct {
	// Command is the command to run remotely. This is executed as if
	// it were a shell command, so you are expected to do any shell escaping
	// necessary.
	Command string

	// Stdin specifies the process's standard input. If Stdin is
	// nil, the process reads from an empty bytes.Buffer.
	Stdin io.Reader

	// Stdout and Stderr represent the process's standard output and
	// error.
	//
	// If either is nil, it will be set to ioutil.Discard.
	Stdout io.Writer
	Stderr io.Writer

	// Once Wait returns, his will contain the exit code of the process.
	exitStatus int

	// Internal fields
	exitCh chan struct{}

	// err is used to store any error reported by the Communicator during
	// execution.
	err error

	// This thing is a mutex, lock when making modifications concurrently
	sync.Mutex
}

// Init must be called by the Communicator before executing the command.
func (c *Cmd) Init() {
	c.Lock()
	defer c.Unlock()

	c.exitCh = make(chan struct{})
}

// SetExitStatus stores the exit status of the remote command as well as any
// communicator related error. SetExitStatus then unblocks any pending calls
// to Wait.
// This should only be called by communicators executing the remote.Cmd.
func (c *Cmd) SetExitStatus(status int, err error) {
	c.Lock()
	defer c.Unlock()

	c.exitStatus = status
	c.err = err

	close(c.exitCh)
}

// Wait waits for the remote command to complete.
// Wait may return an error from the communicator, or an ExitError if the
// process exits with a non-zero exit status.
func (c *Cmd) Wait() error {
	<-c.exitCh

	c.Lock()
	defer c.Unlock()

	if c.err != nil || c.exitStatus != 0 {
		return &ExitError{
			Command:    c.Command,
			ExitStatus: c.exitStatus,
			Err:        c.err,
		}
	}

	return nil
}

// ExitError is returned by Wait to indicate and error executing the remote
// command, or a non-zero exit status.
type ExitError struct {
	Command    string
	ExitStatus int
	Err        error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("error executing %q: %v", e.Command, e.Err)
	}
	return fmt.Sprintf("%q exit status: %d", e.Command, e.ExitStatus)
}
