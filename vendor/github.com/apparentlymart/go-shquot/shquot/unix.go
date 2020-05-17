package shquot

import (
	"strings"
)

// UnixTerminal produces a quoting function that prepares a command line to
// pass through a Unix-style terminal driver.
//
// On Unix systems we usually execute commands programmatically by directly
// executing them, rather than by delivering them to a shell through a terminal.
// However, in some rare cases a command line must be delivered as if typed
// manually at a terminal.
//
// This function constructs a filter function that escapes control characters
// using the ^V control character, which is the default representation of
// the "literal next" (lnext) Terminal command. Since the terminal driver is
// only an intermediary on the way to some other shell, you must pass the
// quoting function for that final shell in order to obtain a function that
// applies both levels of quoting/escaping at once.
//
// You don't need to use this function unless you are delivering a command
// line through a terminal driver. Such situations include programmatically
// typing commands into the console of a virtual machine via a virtualized
// keyboard, sending automated keystrokes to a real server via a serial console
// interface, or delivering a command to an interactive shell through a
// pseudo-terminal (pty). If you can avoid doing these things by directly
// executing the shell in a non-interactive mode, please do.
//
// Note that the lnext command can potentially be remapped to a different
// control character using stty, in which case this function's result will
// be misinterpreted. Use this function only with terminals using the default
// ^V mapping.
//
// The "lnext" command is a Minix extension and not part of the POSIX standard,
// but is in practice implemented on most modern Unix-style operating systems,
// including Linux. Additionally, most modern Unix shells accept interactive
// input in "raw" mode and thus any control characters are handled directly by
// the shell rather than by the terminal driver, and many treat lnext in the
// same way as the terminal driver would.
func UnixTerminal(wrapped Q) Q {
	return func(cmdline []string) string {
		s := wrapped(cmdline)

		var to strings.Builder
		litLen := 0
		remain := s
		for len(remain) > 0 {
			var this byte
			this, remain = remain[0], remain[1:]

			if this < 32 || this == 127 {
				// If it's an ASCII control character then we'll emit ^V to
				// try to escape it. If we've skipped over any literal
				// characters then we need to emit them first.
				if litLen > 0 {
					to.WriteString(s[:litLen])
					litLen = 0
				}
				to.WriteByte(0x14) // Ctrl+V; default control code for "lnext"
				to.WriteByte(this)
				s = remain // resync "s" to avoid treating this character as a literal on a future loop
				continue
			}
			litLen++
		}
		// If there's anything left in "s" at this point then it's trailing literal
		// characters.
		if len(s) > 0 {
			to.WriteString(s)
		}
		return to.String()
	}
}
