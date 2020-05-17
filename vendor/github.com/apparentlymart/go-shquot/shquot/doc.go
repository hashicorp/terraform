// Package shquot contains various functions for quoting sequences of
// command arguments for literal interpretation by different shells and other
// similar intermediaries that process incoming command lines.
//
// The functions all have the same signature, defined as type Q in this package,
// taking an array of arguments in the usual style passed to "execve" on a
// Unix/POSIX system.
//
// While calling "execve" directly is always preferable to avoid
// misinterpretation by intermediaries, sometimes such preprocessing cannot
// be avoided. For example, remote command execution protocols like SSH often
// expect a single string to be interpreted by a shell.
//
// Since each shell or intermediary has different details, it's important to
// select the correct quoting function for the target system or else the
// result may be misinterpreted.
//
// The goal of functions in this package is to cause a command line to be
// interpreted as if it were passed to the "execve" C function, bypassing any
// special behaviors of intermediaries such as wildcard expansion, alias
// expansion, pipelines, etc. If any shell behaviors are accessible through
// crafted input to the corresponding function then that's always considered
// to be a bug unless specifically noted in the documentation for that function.
package shquot
