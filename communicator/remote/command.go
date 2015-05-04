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

	// This will be set to true when the remote command has exited. It
	// shouldn't be set manually by the user, but there is no harm in
	// doing so.
	Exited bool

	// Once Exited is true, this will contain the exit code of the process.
	ExitStatus int

	// Internal fields
	exitCh chan struct{}

	// This thing is a mutex, lock when making modifications concurrently
	sync.Mutex
}

// SetExited is a helper for setting that this process is exited. This
// should be called by communicators who are running a remote command in
// order to set that the command is done.
func (r *Cmd) SetExited(status int) {
	r.Lock()
	defer r.Unlock()

	if r.exitCh == nil {
		r.exitCh = make(chan struct{})
	}

	r.Exited = true
	r.ExitStatus = status
	close(r.exitCh)
}

// Wait waits for the remote command to complete.
func (r *Cmd) Wait() {
	// Make sure our condition variable is initialized.
	r.Lock()
	if r.exitCh == nil {
		r.exitCh = make(chan struct{})
	}
	r.Unlock()

	<-r.exitCh
}
