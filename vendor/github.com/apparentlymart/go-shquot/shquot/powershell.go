package shquot

import (
	"strings"
)

var powerShellReplace = strings.NewReplacer(
	"`", "``",
	`"`, "`\"",
	"$", "`$",
	"\x00", "`0",
	"\x07", "`a",
	"\x08", "`b",
	"\x1f", "`e",
	"\x0c", "`f",
	"\n", "`n",
	"\r", "`r",
	"\t", "`t",
	"\v", "`v",
)

// ViaPowerShell wraps a "Split" quoting function to create a normal quoting
// function that produces a command string that can be passed to PowerShell to
// run the given command line as an external process.
//
// PowerShell uses a single string for all of the arguments, which must itself
// be quoted in a manner suitable for the program being run. The wrapped
// split-quoting function provides the quoting for the arguments.
//
// For example, if using PowerShell to run a program that uses the MSVC runtime's
// argument parsing functionality on Windows, use ViaPowerShell(WindowsArgvSplit)
// to get a suitably-quoted command line.
//
// This function uses the PowerShell Start-Process cmdlet to force the
// command to be interpreted as an external program rather than as a cmdlet
// or other PowerShell command type.
func ViaPowerShell(wrapped QS) Q {
	return func(cmdline []string) string {
		if len(cmdline) == 0 {
			return ""
		}
		cmd, args := wrapped(cmdline)

		var buf strings.Builder
		buf.WriteString("& Start-Process -FilePath ")
		powerShellQuoted(cmd, &buf)
		if len(args) > 0 {
			buf.WriteString(" -ArgumentList ")
			powerShellQuoted(args, &buf)
		}
		return buf.String()
	}
}

func powerShellQuoted(a string, buf *strings.Builder) {
	buf.WriteByte('"')
	powerShellReplace.WriteString(buf, a)
	buf.WriteByte('"')
}
