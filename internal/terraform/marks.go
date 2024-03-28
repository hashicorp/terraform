// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
)

// valueMarksEqual compares the marks of 2 cty.Values for equality.
func valueMarksEqual(a, b cty.Value) bool {
	_, aMarks := a.UnmarkDeepWithPaths()
	_, bMarks := b.UnmarkDeepWithPaths()
	return marksEqual(aMarks, bMarks)
}

// marksEqual compares 2 unordered sets of PathValue marks for equality, with
// the comparison using the cty.PathValueMarks.Equal method.
func marksEqual(a, b []cty.PathValueMarks) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	if len(a) != len(b) {
		return false
	}

	less := func(s []cty.PathValueMarks) func(i, j int) bool {
		return func(i, j int) bool {
			// the sort only needs to be consistent, so use the GoString format
			// to get a comparable value
			return fmt.Sprintf("%#v", s[i]) < fmt.Sprintf("%#v", s[j])
		}
	}

	sort.Slice(a, less(a))
	sort.Slice(b, less(b))

	for i := 0; i < len(a); i++ {
		if !a[i].Equal(b[i]) {
			return false
		}
	}

	return true
}

// Remove duplicate PathValueMarks from the slice.
// When adding marks from a resource schema to a value, most of the time there
// will be duplicates from a prior value already in the list of marks. While
// MarkWithPaths will accept duplicates, we need to be able to easily compare
// the PathValueMarks within this package too.
func dedupePathValueMarks(m []cty.PathValueMarks) []cty.PathValueMarks {
	var res []cty.PathValueMarks
	// we'll use a GoString format key to differentiate PathValueMarks, since
	// the Path portion is not automagically comparable.
	seen := make(map[string]bool)

	for _, pvm := range m {
		key := fmt.Sprintf("%#v", pvm)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = true
		res = append(res, pvm)
	}

	return res
}
