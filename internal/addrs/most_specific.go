// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

// MostSpecific returns the most specific of the given candidates, which is the
// one contained by all of the others according to contains, which returns
// true if its first argument contains its second.
//
// Each candidate replaces the result so far if the result contains it but it
// doesn't contain the result, so when the candidates aren't all nested within
// each other, earlier candidates take precedence. Callers which need a
// deterministic result must give the candidates in a deterministic order.
//
// The second return value is false if there are no candidates.
func MostSpecific[T any](candidates []T, contains func(a, b T) bool) (T, bool) {
	var best T
	if len(candidates) == 0 {
		return best, false
	}

	best = candidates[0]
	for _, candidate := range candidates[1:] {
		if contains(best, candidate) && !contains(candidate, best) {
			best = candidate
		}
	}
	return best, true
}
