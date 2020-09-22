// The panicwrap package provides functions for capturing and handling
// panics in your application. It does this by re-executing the running
// application and monitoring stderr for any panics. At the same time,
// stdout/stderr/etc. are set to the same values so that data is shuttled
// through properly, making the existence of panicwrap mostly transparent.
//
// Panics are only detected when the subprocess exits with a non-zero
// exit status, since this is the only time panics are real. Otherwise,
// "panic-like" output is ignored.
package panicwrap

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	DEFAULT_COOKIE_KEY = "cccf35992f8f3cd8d1d28f0109dd953e26664531"
	DEFAULT_COOKIE_VAL = "7c28215aca87789f95b406b8dd91aa5198406750"
)

// HandlerFunc is the type called when a panic is detected.
type HandlerFunc func(string)

// WrapConfig is the configuration for panicwrap when wrapping an existing
// binary. To get started, in general, you only need the BasicWrap function
// that will set this up for you. However, for more customizability,
// WrapConfig and Wrap can be used.
type WrapConfig struct {
	// Handler is the function called when a panic occurs.
	Handler HandlerFunc

	// The cookie key and value are used within environmental variables
	// to tell the child process that it is already executing so that
	// wrap doesn't re-wrap itself.
	CookieKey   string
	CookieValue string

	// If true, the panic will not be mirrored to the configured writer
	// and will instead ONLY go to the handler. This lets you effectively
	// hide panics from the end user. This is not recommended because if
	// your handler fails, the panic is effectively lost.
	HidePanic bool

	// The amount of time that a process must exit within after detecting
	// a panic header for panicwrap to assume it is a panic. Defaults to
	// 300 milliseconds.
	DetectDuration time.Duration

	// The writer to send the stderr to. If this is nil, then it defaults
	// to os.Stderr.
	Writer io.Writer

	// The writer to send stdout to. If this is nil, then it defaults to
	// os.Stdout.
	Stdout io.Writer

	// Catch and igore these signals in the parent process, let the child
	// handle them gracefully.
	IgnoreSignals []os.Signal

	// Catch these signals in the parent process and manually forward
	// them to the child process. Some signals such as SIGINT are usually
	// sent to the entire process group so setting it isn't necessary. Other
	// signals like SIGTERM are only sent to the parent process and need
	// to be forwarded. This defaults to empty.
	ForwardSignals []os.Signal
}

// BasicWrap calls Wrap with the given handler function, using defaults
// for everything else. See Wrap and WrapConfig for more information on
// functionality and return values.
func BasicWrap(f HandlerFunc) (int, error) {
	return Wrap(&WrapConfig{
		Handler: f,
	})
}

// Wrap wraps the current executable in a handler to catch panics. It
// returns an error if there was an error during the wrapping process.
// If the error is nil, then the int result indicates the exit status of the
// child process. If the exit status is -1, then this is the child process,
// and execution should continue as normal. Otherwise, this is the parent
// process and the child successfully ran already, and you should exit the
// process with the returned exit status.
//
// This function should be called very very early in your program's execution.
// Ideally, this runs as the first line of code of main.
//
// Once this is called, the given WrapConfig shouldn't be modified or used
// any further.
func Wrap(c *WrapConfig) (int, error) {
	if c.Handler == nil {
		return -1, errors.New("Handler must be set")
	}

	if c.DetectDuration == 0 {
		c.DetectDuration = 300 * time.Millisecond
	}

	if c.Writer == nil {
		c.Writer = os.Stderr
	}

	// If we're already wrapped, exit out.
	if Wrapped(c) {
		return -1, nil
	}

	// Get the path to our current executable
	exePath, err := os.Executable()
	if err != nil {
		return -1, err
	}

	// Pipe the stderr so we can read all the data as we look for panics
	stderr_r, stderr_w := io.Pipe()

	// doneCh is closed when we're done, signaling any other goroutines
	// to end immediately.
	doneCh := make(chan struct{})

	// panicCh is the channel on which the panic text will actually be
	// sent.
	panicCh := make(chan string)

	// On close, make sure to finish off the copying of data to stderr
	defer func() {
		defer close(doneCh)
		stderr_w.Close()
		<-panicCh
	}()

	// Start the goroutine that will watch stderr for any panics
	go trackPanic(stderr_r, c.Writer, c.DetectDuration, panicCh)

	// Create the writer for stdout that we're going to use
	var stdout_w io.Writer = os.Stdout
	if c.Stdout != nil {
		stdout_w = c.Stdout
	}

	// Build a subcommand to re-execute ourselves. We make sure to
	// set the environmental variable to include our cookie. We also
	// set stdin/stdout to match the config. Finally, we pipe stderr
	// through ourselves in order to watch for panics.
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = append(os.Environ(), c.CookieKey+"="+c.CookieValue)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout_w
	cmd.Stderr = stderr_w

	// Windows doesn't support this, but on other platforms pass in
	// the original file descriptors so they can be used.
	if runtime.GOOS != "windows" {
		cmd.ExtraFiles = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	}

	if err := cmd.Start(); err != nil {
		return 1, err
	}

	// Listen to signals and capture them forever. We allow the child
	// process to handle them in some way.
	sigCh := make(chan os.Signal)
	fwdSigCh := make(chan os.Signal)
	if len(c.IgnoreSignals) == 0 {
		c.IgnoreSignals = []os.Signal{os.Interrupt}
	}
	signal.Notify(sigCh, c.IgnoreSignals...)
	signal.Notify(fwdSigCh, c.ForwardSignals...)
	go func() {
		defer signal.Stop(sigCh)
		defer signal.Stop(fwdSigCh)
		for {
			select {
			case <-doneCh:
				return
			case s := <-fwdSigCh:
				if cmd.Process != nil {
					cmd.Process.Signal(s)
				}
			case <-sigCh:
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			// This is some other kind of subprocessing error.
			return 1, err
		}

		exitStatus := 1
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		}

		// Close the writer end so that the tracker goroutine ends at some point
		stderr_w.Close()

		// Wait on the panic data
		panicTxt := <-panicCh
		if panicTxt != "" {
			if !c.HidePanic {
				c.Writer.Write([]byte(panicTxt))
			}

			c.Handler(panicTxt)
		}

		return exitStatus, nil
	}

	return 0, nil
}

// Wrapped checks if we're already wrapped according to the configuration
// given.
//
// It must be only called once with a non-nil configuration as it unsets
// the environment variable it uses to check if we are already wrapped.
// This prevents false positive if your program tries to execute itself
// recursively.
//
// Wrapped is very cheap and can be used early to short-circuit some pre-wrap
// logic your application may have.
//
// If the given configuration is nil, then this will return a cached
// value of Wrapped. This is useful because Wrapped is usually called early
// to verify a process hasn't been wrapped before wrapping. After this,
// the value of Wrapped hardly changes and is process-global, so other
// libraries can check with Wrapped(nil).
func Wrapped(c *WrapConfig) bool {
	if c == nil {
		return wrapCache.Load().(bool)
	}

	if c.CookieKey == "" {
		c.CookieKey = DEFAULT_COOKIE_KEY
	}

	if c.CookieValue == "" {
		c.CookieValue = DEFAULT_COOKIE_VAL
	}

	// If the cookie key/value match our environment, then we are the
	// child, so just exit now and tell the caller that we're the child
	result := os.Getenv(c.CookieKey) == c.CookieValue
	if result {
		os.Unsetenv(c.CookieKey)
	}
	wrapCache.Store(result)
	return result
}

// wrapCache is the cached value for Wrapped when called with nil
var wrapCache atomic.Value

func init() {
	wrapCache.Store(false)
}

// trackPanic monitors the given reader for a panic. If a panic is detected,
// it is outputted on the result channel. This will close the channel once
// it is complete.
func trackPanic(r io.Reader, w io.Writer, dur time.Duration, result chan<- string) {
	defer close(result)

	var panicTimer <-chan time.Time
	panicBuf := new(bytes.Buffer)
	panicHeaders := [][]byte{
		[]byte("panic:"),
		[]byte("fatal error: fault"),
	}
	panicType := -1

	tempBuf := make([]byte, 2048)
	for {
		var buf []byte
		var n int

		if panicTimer == nil && panicBuf.Len() > 0 {
			// We're not tracking a panic but the buffer length is
			// greater than 0. We need to clear out that buffer, but
			// look for another panic along the way.

			// First, remove the previous panic header so we don't loop
			w.Write(panicBuf.Next(len(panicHeaders[panicType])))

			// Next, assume that this is our new buffer to inspect
			n = panicBuf.Len()
			buf = make([]byte, n)
			copy(buf, panicBuf.Bytes())
			panicBuf.Reset()
		} else {
			var err error
			buf = tempBuf
			n, err = r.Read(buf)
			if n <= 0 && err == io.EOF {
				if panicBuf.Len() > 0 {
					// We were tracking a panic, assume it was a panic
					// and return that as the result.
					result <- panicBuf.String()
				}

				return
			}
		}

		if panicTimer != nil {
			// We're tracking what we think is a panic right now.
			// If the timer ended, then it is not a panic.
			isPanic := true
			select {
			case <-panicTimer:
				isPanic = false
			default:
			}

			// No matter what, buffer the text some more.
			panicBuf.Write(buf[0:n])

			if !isPanic {
				// It isn't a panic, stop tracking. Clean-up will happen
				// on the next iteration.
				panicTimer = nil
			}

			continue
		}

		panicType = -1
		flushIdx := n
		for i, header := range panicHeaders {
			idx := bytes.Index(buf[0:n], header)
			if idx >= 0 {
				panicType = i
				flushIdx = idx
				break
			}
		}

		// Flush to stderr what isn't a panic
		w.Write(buf[0:flushIdx])

		if panicType == -1 {
			// Not a panic so just continue along
			continue
		}

		// We have a panic header. Write we assume is a panic os far.
		panicBuf.Write(buf[flushIdx:n])
		panicTimer = time.After(dur)
	}
}
