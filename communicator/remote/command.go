package remote

import (
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

// Err returns any communicator related error.
func (c *Cmd) Err() error {
	c.Lock()
	defer c.Unlock()

	return c.err
}

// ExitStatus returns the exit status of the remote command
func (c *Cmd) ExitStatus() int {
	c.Lock()
	defer c.Unlock()

	return c.exitStatus
}

// Wait waits for the remote command to complete.
func (c *Cmd) Wait() {
	<-c.exitCh
}
