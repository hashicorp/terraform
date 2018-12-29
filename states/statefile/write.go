package statefile

import (
	"io"
)

// Write writes the given state to the given writer in the current state
// serialization format.
func Write(s *File, w io.Writer) error {
	diags := writeStateV4(s, w)
	return diags.Err()
}
