// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
