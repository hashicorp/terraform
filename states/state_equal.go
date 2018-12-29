package states

import (
	"reflect"
)

// Equal returns true if the receiver is functionally equivalent to other,
// including any ephemeral portions of the state that would not be included
// if the state were saved to files.
//
// To test only the persistent portions of two states for equality, instead
// use statefile.StatesMarshalEqual.
func (s *State) Equal(other *State) bool {
	// For the moment this is sufficient, but we may need to do something
	// more elaborate in future if we have any portions of state that require
	// more sophisticated comparisons.
	return reflect.DeepEqual(s, other)
}
