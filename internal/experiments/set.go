// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package experiments

// Set is a collection of experiments where every experiment is either a member
// or not.
type Set map[Experiment]struct{}

// NewSet constructs a new Set with the given experiments as its initial members.
func NewSet(exps ...Experiment) Set {
	ret := make(Set)
	for _, exp := range exps {
		ret.Add(exp)
	}
	return ret
}

// SetUnion constructs a new Set containing the members of all of the given
// sets.
func SetUnion(sets ...Set) Set {
	ret := make(Set)
	for _, set := range sets {
		for exp := range set {
			ret.Add(exp)
		}
	}
	return ret
}

// Add inserts the given experiment into the set.
//
// If the given experiment is already present then this is a no-op.
func (s Set) Add(exp Experiment) {
	s[exp] = struct{}{}
}

// Remove takes the given experiment out of the set.
//
// If the given experiment not already present then this is a no-op.
func (s Set) Remove(exp Experiment) {
	delete(s, exp)
}

// Has tests whether the given experiment is in the receiving set.
func (s Set) Has(exp Experiment) bool {
	_, ok := s[exp]
	return ok
}
