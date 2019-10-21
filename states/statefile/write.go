package statefile

import (
	"io"

	tfversion "github.com/hashicorp/terraform/version"
)

// Write writes the given state to the given writer in the current state
// serialization format.
func Write(s *File, w io.Writer) error {
	// Always record the current terraform version in the state.
	s.TerraformVersion = tfversion.SemVer

	diags := writeStateV4(s, w)
	return diags.Err()
}
