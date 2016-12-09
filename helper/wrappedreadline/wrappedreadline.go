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
	"runtime"

	"github.com/chzyer/readline"

	"github.com/hashicorp/terraform/helper/wrappedstreams"
)

// Override overrides the values in readline.Config that need to be
// set with wrapped values.
func Override(cfg *readline.Config) *readline.Config {
	cfg.Stdin = wrappedstreams.Stdin()
	cfg.Stdout = wrappedstreams.Stdout()
	cfg.Stderr = wrappedstreams.Stderr()

	cfg.FuncGetWidth = TerminalWidth
	cfg.FuncIsTerminal = IsTerminal

	rm := RawMode{StdinFd: int(wrappedstreams.Stdin().Fd())}
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
	return readline.IsTerminal(int(wrappedstreams.Stdin().Fd())) &&
		(readline.IsTerminal(int(wrappedstreams.Stdout().Fd())) ||
			readline.IsTerminal(int(wrappedstreams.Stderr().Fd())))
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
	StdinFd int

	state *readline.State
}

func (r *RawMode) Enter() (err error) {
	r.state, err = readline.MakeRaw(r.StdinFd)
	return err
}

func (r *RawMode) Exit() error {
	if r.state == nil {
		return nil
	}

	return readline.Restore(r.StdinFd, r.state)
}
