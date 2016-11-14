// wrappedreadline is a package that has helpers for interacting with
// readline from a panicwrap executable.
//
// panicwrap overrides the standard file descriptors so that the child process
// no longer looks like a TTY. The helpers here access the extra file descriptors
// passed by panicwrap to fix that.
//
// panicwrap should be checked for with panicwrap.Wrapped before using this
// librar, since this library won't adapt if the binary is not wrapped.
package wrappedreadline

import (
	"os"
	"runtime"

	"github.com/chzyer/readline"
)

// These are the file descriptor numbers for the original stdin, stdout, stderr
// streams from the parent process.
const (
	StdinFd  = 3
	StdoutFd = 4
	StderrFd = 5
)

// These are the *os.File values for the standard streams.
var (
	Stdin  = os.NewFile(uintptr(StdinFd), "stdin")
	Stdout = os.NewFile(uintptr(StdoutFd), "stdout")
	Stderr = os.NewFile(uintptr(StderrFd), "stderr")
)

// Override overrides the values in readline.Config that need to be
// set with wrapped values.
func Override(cfg *readline.Config) *readline.Config {
	cfg.Stdin = Stdin
	cfg.Stdout = Stdout
	cfg.Stderr = Stderr

	cfg.FuncGetWidth = TerminalWidth
	cfg.FuncIsTerminal = IsTerminal

	var rm RawMode
	cfg.FuncMakeRaw = rm.Enter
	cfg.FuncExitRaw = rm.Exit

	return cfg
}

// IsTerminal determines if this process is attached to a TTY.
func IsTerminal() bool {
	// Windows is always a terminal
	if runtime.GOOS == "windows" {
		return true
	}

	// Same implementation as readline but with our custom fds
	return readline.IsTerminal(StdinFd) && (readline.IsTerminal(StdoutFd) || readline.IsTerminal(StderrFd))
}

// TerminalWidth gets the terminal width in characters.
func TerminalWidth() int {
	if runtime.GOOS == "windows" {
		return readline.GetScreenWidth()
	}

	return getWidth()
}

// RawMode is a helper for entering and exiting raw mode.
type RawMode struct {
	state *readline.State
}

func (r *RawMode) Enter() (err error) {
	r.state, err = readline.MakeRaw(StdinFd)
	return err
}

func (r *RawMode) Exit() error {
	if r.state == nil {
		return nil
	}

	return readline.Restore(StdinFd, r.state)
}
