// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/zclconf/go-cty/cty"
)

// PathsWithMark produces a list of paths identified as having a specified
// mark in a given set of [cty.PathValueMarks] that presumably resulted from
// deeply-unmarking a [cty.Value].
//
// This is for situations where a subsystem needs to give special treatment
// to one specific mark value, as opposed to just handling all marks
// generically as cty operations would. The second return value is a
// subset of the given [cty.PathValueMarks] values which contained marks
// other than the one requested, so that a caller that can't preserve other
// marks at all can more easily return an error explaining that.
func PathsWithMark(pvms []cty.PathValueMarks, wantMark any) (withWanted []cty.Path, withOthers []cty.PathValueMarks) {
	if len(pvms) == 0 {
		// No-allocations path for the common case where there are no marks at all.
		return nil, nil
	}

	for _, pvm := range pvms {
		pathHasMark := false
		pathHasOtherMarks := false
		for mark := range pvm.Marks {
			switch wantMark.(type) {
			case valueMark, string:
				if mark == wantMark {
					pathHasMark = true
				} else {
					pathHasOtherMarks = true
				}

			// For data marks we check if a mark of the type exists
			case DeprecationMark:
				if _, ok := mark.(DeprecationMark); ok {
					pathHasMark = true
				} else {
					pathHasOtherMarks = true
				}

			default:
				panic(fmt.Sprintf("unexpected mark type %T", wantMark))
			}
		}

		if pathHasMark {
			withWanted = append(withWanted, pvm.Path)
		}

		if pathHasOtherMarks {
			withOthers = append(withOthers, pvm)
		}
	}

	return withWanted, withOthers
}

// RemoveAll take a series of PathValueMarks and removes the unwanted mark from
// all paths. Paths with no remaining marks will be removed entirely. The
// PathValuesMarks passed in are not cloned, and RemoveAll will modify the
// original values, so the prior set of marks should not be retained for use.
func RemoveAll(pvms []cty.PathValueMarks, remove any) []cty.PathValueMarks {
	if len(pvms) == 0 {
		// No-allocations path for the common case where there are no marks at all.
		return nil
	}

	var res []cty.PathValueMarks

	for _, pvm := range pvms {
		switch remove.(type) {
		case valueMark, string:
			delete(pvm.Marks, remove)

		case DeprecationMark:
			// We want to delete all marks of this type
			for mark := range pvm.Marks {
				if _, ok := mark.(DeprecationMark); ok {
					delete(pvm.Marks, mark)
				}
			}

		default:
			panic(fmt.Sprintf("unexpected mark type %T", remove))
		}
		if len(pvm.Marks) > 0 {
			res = append(res, pvm)
		}
	}

	return res
}

// MarkPaths transforms the given value by marking each of the given paths
// with the given mark value.
func MarkPaths(val cty.Value, mark any, paths []cty.Path) cty.Value {
	if len(paths) == 0 {
		// No-allocations path for the common case where there are no marked paths at all.
		return val
	}

	// For now we'll use cty's slightly lower-level function to achieve this
	// result. This is a little inefficient due to an additional dynamic
	// allocation for the intermediate data structure, so if that becomes
	// a problem in practice then we may wish to write a more direct
	// implementation here.
	markses := make([]cty.PathValueMarks, len(paths))
	marks := cty.NewValueMarks(mark)
	for i, path := range paths {
		markses[i] = cty.PathValueMarks{
			Path:  path,
			Marks: marks,
		}
	}
	return val.MarkWithPaths(markses)
}

// MarksEqual compares 2 unordered sets of PathValue marks for equality, with
// the comparison using the cty.PathValueMarks.Equal method.
func MarksEqual(a, b []cty.PathValueMarks) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	if len(a) != len(b) {
		return false
	}

	less := func(s []cty.PathValueMarks) func(i, j int) bool {
		return func(i, j int) bool {
			cmp := strings.Compare(format.CtyPath(s[i].Path), format.CtyPath(s[j].Path))

			switch {
			case cmp < 0:
				return true
			case cmp > 0:
				return false
			}
			// the sort only needs to be consistent, so use the GoString format
			// to get a comparable value
			return s[i].Marks.GoString() < s[j].Marks.GoString()
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
