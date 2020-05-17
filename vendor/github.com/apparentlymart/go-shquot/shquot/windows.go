package shquot

import (
	"strings"
)

// WindowsArgv quotes arguments using conventions that lead to correct parsing
// by both the Windows API function CommandLineToArgvW and the parse_cmdline
// function in the Microsoft Visual C++ runtime library.
//
// On Windows the final parsing of a command line string is the responsibility
// of the application itself, and so an application may employ any strategy
// it wishes to parse the command line.
//
// In practice though, most command line Windows applications receive their
// arguments via the argv/argc parameters to function main, which contain the
// result of calling parse_cmdline.
//
// Windows GUI applications receive a command line string as one argument to
// WinMain, but it has already been partially processed to remove the first
// argument that is conventionally the program name. Windows applications may
// instead choose to call GetCommandLine to obtain the full original string
// and then process it with the API function CommandLineToArgvW to obtain
// an argv/argc pair.
//
// The parse_cmdline and CommandLineToArgvW implementations do not have
// identical behavior, but their behavior is compatible enough that this
// function can produce a string that can be processed successfully by both.
// For applications that use neither of these mechanisms there is no guarantee
// that any particular quoting scheme will work.
//
// In particular note that programs that are not written in C or C++ (e.g.
// batch files, cmd scripts, Windows Script Host programs, .NET Framework
// applications, etc) are likely to have divergent processing rules that this
// function cannot guarantee to support.
//
// There is one important caveat with this function: neither of the "argv"
// construction approaches supports escaping of quotes in the first argument,
// so quote characters cannot be reliably supported there. If any are found,
// this function will strip them out. In practice a quote character is never
// valid in a program name on Windows and so this should have no practical
// impact, but if you wish to detect that scenario you can first call
// WindowsArgvValid to determine whether a particular command line slice can be
// losslessly encoded by WindowsArgv.
func WindowsArgv(cmdline []string) string {
	if len(cmdline) == 0 {
		return ""
	}

	// The rules here were derived from the information at the following sources,
	// since the Windows API and Visual C++ Runtime documentation are both
	// incomplete:
	// http://www.windowsinspired.com/how-a-windows-programs-splits-its-command-line-into-individual-arguments/
	// http://daviddeley.com/autohotkey/parameters/parameters.htm

	var buf strings.Builder
	windowsArgvFirst(cmdline[0], &buf)
	for _, a := range cmdline[1:] {
		buf.WriteByte(' ')
		windowsArgvSingle(a, &buf)
	}
	return buf.String()
}

// WindowsArgvSplit is a variant of WindowsArgv that quotes only the arguments
// in the given command line -- that is, indices 1 and greater in the given
// slice -- and just returns the command from index 0 verbatim to be quoted
// by another layer.
//
// This is useful for Windows-style process-starting APIs where the command
// itself is isolated but the arguments are provided as a single, already-quoted
// string.
func WindowsArgvSplit(cmdline []string) (cmd, args string) {
	if len(cmdline) == 0 {
		return "", ""
	}

	cmd = cmdline[0]
	var buf strings.Builder
	for i, a := range cmdline[1:] {
		if i > 0 {
			buf.WriteByte(' ')
		}
		windowsArgvSingle(a, &buf)
	}
	return cmd, buf.String()
}

// WindowsArgvValid is a helper for use alongside WindowsArgv to deal with the
// fact that the commonly-used Windows command line parsers do not support
// escaping of quotes in the first element of the command line.
//
// This function returns false if the first element of the given cmdline contains
// quote characters, or true otherwise. If this function returns false then
// the return value of WindowsArgv for the same cmdline may be lossy.
func WindowsArgvValid(cmdline []string) bool {
	if len(cmdline) == 0 {
		return true
	}

	return !strings.ContainsRune(cmdline[0], '"')
}

func windowsArgvFirst(a string, to *strings.Builder) {
	// The first argument in a string processed by either parse_cmdline or
	// CommandLineToArgvW is handled using a distinct set of rules that differ
	// significantly between the two implementations, but if we ensure the
	// string always starts and ends with a quote mark and contains no
	// intervening quotes then it can be processed by both.
	// Since there is no mechanism to escape quotes, they are just stripped
	// out altogether by this function.
	to.WriteByte('"')
	for {
		quot := strings.Index(a, `"`)
		if quot == -1 { // No quotes left, so everything else is literal and we're done,
			to.WriteString(a)
			break
		}
		to.WriteString(a[:quot])
		a = a[quot+1:] // skip over quote altogether, stripping it out
	}
	to.WriteByte('"')
}

func windowsArgvSingle(a string, to *strings.Builder) {
	if len(a) > 0 && !strings.ContainsAny(a, " \t\n\v\"") {
		// No quoting required, then.
		to.WriteString(a)
		return
	}

	to.WriteByte('"')
	bs := 0
	for _, c := range a {
		switch c {
		case '\\':
			bs++
			continue
		case '"':
			// All of the backslashes we saw so far must be escaped, and then
			// we need one more backslash for the quote character.
			to.WriteString(strings.Repeat("\\", bs*2+1))
			to.WriteRune(c)
			bs = 0
		default:
			// If we encounter anything other than a quote or a backslash
			// then any preceding backslashes we've seen are _not_ special and
			// so we must write them out literally first.
			if bs > 0 {
				to.WriteString(strings.Repeat("\\", bs))
			}
			to.WriteRune(c)
			bs = 0
		}
	}
	// If any backslashes are pending once we exit then we need to double them
	// all up so that the closing quote will _not_ be interpreted as an escape.
	if bs > 0 {
		to.WriteString(strings.Repeat("\\", bs*2))
	}
	to.WriteByte('"')
}

// WindowsCmdExe produces a quoting function that prepares a command line to
// pass through the Windows command interpreter cmd.exe.
//
// Since cmd.exe is just an intermediary, the caller must provide another
// quoting function that deals with the subsequent layer of quoting. On
// Windows most of the command line processing is actually delegated to the
// application itself rather than the command interpreter, and so which
// wrapped quoting function to select depends on the target program.
// Most modern command line applications use the CommandLineToArgvW function
// for argument processing, and its escaping rules are implemented by
// WindowsArgv in this package.
//
// Note that this extra level of quoting is necessary only for command lines
// that will pass through the command interpreter, such as generated command
// scripts. If you're calling the Windows CreateProcess API directly then you
// must not apply cmd.exe quoting, or the result will be incorrectly parsed.
//
// This function cannot prevent expansion of Console Aliases, so the result
// is safe to run only in a command interpreter with no aliases configured.
func WindowsCmdExe(wrapped Q) Q {
	r := strings.NewReplacer(
		"(", "^(",
		")", "^)",
		"%", "^%",
		"!", "^!",
		"^", "^^",
		`"`, `^"`,
		"<", "^<",
		">", "^>",
		"&", "^&",
		"|", "^|",
		"\r\n", "^\r\n\r\n",
		"\n", "^\n\n",
	)
	return func(cmdline []string) string {
		s := wrapped(cmdline)
		return r.Replace(s)
	}
}
