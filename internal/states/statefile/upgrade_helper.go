// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statefile

import (
	"maps"
)

// shallowCopySlice produces a new slice with the same length as s that
// contains shallow copies of the elements of s.
func shallowCopySlice[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	ret := make(S, len(s))
	copy(ret, s)
	return ret
}

// shallowCopySlice produces a new map with the same keys as m that all
// map to shallow copies of the corresponding elements in m.
func shallowCopyMap[M ~map[K]V, K comparable, V any](m M) M {
	if m == nil {
		return nil
	}
	ret := make(M, len(m))
	maps.Copy(ret, m)
	return ret
}
