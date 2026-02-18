// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package states

// Equal returns true if the receiver is functionally equivalent to other,
// including any ephemeral portions of the state that would not be included
// if the state were saved to files.
//
// To test only the persistent portions of two states for equality, instead
// use statefile.StatesMarshalEqual.
func (s *State) Equal(other *State) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	if !s.RootOutputValuesEqual(other) {
		return false
	}

	if !s.CheckResults.Equal(other.CheckResults) {
		return false
	}

	return s.ManagedResourcesEqual(other)
}

// ManagedResourcesEqual returns true if all of the managed resources tracked
// in the receiver are functionally equivalent to the same tracked in the
// other given state.
//
// This is a more constrained version of Equal that disregards other
// differences, including but not limited to changes to data resources and
// changes to output values.
func (s *State) ManagedResourcesEqual(other *State) bool {
	// First, some accommodations for situations where one of the objects is
	// nil, for robustness since we sometimes use a nil state to represent
	// a prior state being entirely absent.
	if s == other {
		// covers both states being nil, or both states being the exact same
		// object.
		return true
	}

	// Managed resources are technically equal if one state is nil while the
	// other has no resources.
	if s == nil {
		return !other.HasManagedResourceInstanceObjects()
	}
	if other == nil {
		return !s.HasManagedResourceInstanceObjects()
	}

	// If we get here then both states are non-nil.

	if len(s.Modules) != len(other.Modules) {
		return false
	}

	for key, sMod := range s.Modules {
		otherMod, ok := other.Modules[key]
		if !ok {
			return false
		}
		// Something else is wrong if the addresses don't match, but they are
		// definitely not equal
		if !sMod.Addr.Equal(otherMod.Addr) {
			return false
		}

		if len(sMod.Resources) != len(otherMod.Resources) {
			return false
		}

		for key, sRes := range sMod.Resources {
			otherRes, ok := otherMod.Resources[key]
			if !ok {
				return false
			}
			if !sRes.Equal(otherRes) {
				return false
			}
		}
	}

	return true
}

// RootOutputValuesEqual returns true if the root output values tracked in the
// receiver are functionally equivalent to the same tracked in the other given
// state.
func (s *State) RootOutputValuesEqual(s2 *State) bool {
	if s == nil && s2 == nil {
		return true
	}

	if len(s.RootOutputValues) != len(s2.RootOutputValues) {
		return false
	}

	for k, v1 := range s.RootOutputValues {
		v2, ok := s2.RootOutputValues[k]
		if !ok || !v1.Equal(v2) {
			return false
		}
	}

	return true
}
