package states

import (
	"reflect"

	"github.com/hashicorp/terraform/internal/addrs"
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

// ManagedResourcesEqual returns true if all of the managed resources tracked
// in the reciever are functionally equivalent to the same tracked in the
// other given state.
//
// This is a more constrained version of Equal that disregards other
// differences, including but not limited to changes to data resources and
// changes to output values.
func (s *State) ManagedResourcesEqual(other *State) bool {
	// First, some accommodations for situations where one of the objects is
	// nil, for robustness since we sometimes use a nil state to represent
	// a prior state being entirely absent.
	if s == nil && other == nil {
		return true
	}
	if s == nil {
		return !other.HasResources()
	}
	if other == nil {
		return !s.HasResources()
	}

	// If we get here then both states are non-nil.

	// sameManagedResources tests that its second argument has all the
	// resources that the first one does, so we'll call it twice with the
	// arguments inverted to ensure that we'll also catch situations where
	// the second has resources that the first does not.
	return sameManagedResources(s, other) && sameManagedResources(other, s)
}

func sameManagedResources(s1, s2 *State) bool {
	for _, ms := range s1.Modules {
		for _, rs := range ms.Resources {
			addr := rs.Addr
			if addr.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			otherRS := s2.Resource(addr)
			if !reflect.DeepEqual(rs, otherRS) {
				return false
			}
		}
	}

	return true

}
