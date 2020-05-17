package shquot

import (
	"encoding/json"
)

// Dockerfile produces a string suitable for use as the argument to one of
// the directives RUN, CMD, and ENTRYPOINT in the Dockerfile format.
//
// The "exec form" of all of these commands is just a JSON serialization of
// an array of strings, so this function is just a Q-compatible adapter to
// encoding/json.Marshal.
func Dockerfile(cmdline []string) string {
	buf, err := json.Marshal(cmdline)
	if err != nil {
		// Should never happen, since it should always be possible to
		// produce JSON for a []string.
		panic(err.Error())
	}
	return string(buf)
}
