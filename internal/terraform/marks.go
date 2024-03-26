// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
)

// filterMarks removes any PathValueMarks from marks which cannot be applied to
// the given value. When comparing existing marks to those from a map or other
// dynamic value, we may not have values at the same paths and need to strip
// out irrelevant marks.
func filterMarks(val cty.Value, marks []cty.PathValueMarks) []cty.PathValueMarks {
	var res []cty.PathValueMarks
	for _, mark := range marks {
		// any error here just means the path cannot apply to this value, so we
		// don't need this mark for comparison.
		if _, err := mark.Path.Apply(val); err == nil {
			res = append(res, mark)
		}
	}
	return res
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
// MarkwithPaths will accept duplicates, we need to be able to easily compare
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
