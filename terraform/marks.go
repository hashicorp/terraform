package terraform

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
)

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
