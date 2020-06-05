package versions

import (
	"sort"
)

// List is a slice of Version that implements sort.Interface, and also includes
// some other helper functions.
type List []Version

// Filter removes from the receiver any elements that are not in the given
// set, moving retained elements to lower indices to close any gaps and
// modifying the underlying array in-place. The return value is a slice
// describing the new bounds within the existing backing array. The relative
// ordering of the retained elements is preserved.
//
// The result must always be either the same length or shorter than the
// initial value, so no allocation is required.
//
// As a special case, if the result would be a slice of length zero then a
// nil slice is returned instead, leaving the backing array untouched.
func (l List) Filter(set Set) List {
	writeI := 0

	for readI := range l {
		if set.Has(l[readI]) {
			l[writeI] = l[readI]
			writeI++
		}
	}

	if writeI == 0 {
		return nil
	}
	return l[:writeI:len(l)]
}

// Newest returns the newest version in the list, or Unspecified if the list
// is empty.
//
// Since build metadata does not participate in precedence, it is possible
// that a given list may have multiple equally-new versions; in that case
// Newest will return an arbitrary version from that subset.
func (l List) Newest() Version {
	ret := Unspecified
	for i := len(l) - 1; i >= 0; i-- {
		if l[i].GreaterThan(ret) {
			ret = l[i]
		}
	}
	return ret
}

// NewestInSet is like Filter followed by Newest, except that it does not
// modify the underlying array. This is convenient for the common case of
// selecting the newest version from a set derived from a user-supplied
// constraint.
//
// Similar to Newest, the result is Unspecified if the list is empty or if
// none of the items are in the given set. Also similar to newest, if there
// are multiple newest versions (possibly differentiated only by metadata)
// then one is arbitrarily chosen.
func (l List) NewestInSet(set Set) Version {
	ret := Unspecified
	for i := len(l) - 1; i >= 0; i-- {
		if l[i].GreaterThan(ret) && set.Has(l[i]) {
			ret = l[i]
		}
	}
	return ret
}

// NewestList returns a List containing all of the list items that have the
// highest precedence.
//
// For an already-sorted list, the returned slice is a sub-slice of the
// receiver, sharing the same backing array. For an unsorted list, a new
// array is allocated for the result. For an empty list, the result is always
// nil.
//
// Relative ordering of elements in the receiver is preserved in the output.
func (l List) NewestList() List {
	if len(l) == 0 {
		return nil
	}

	if l.IsSorted() {
		// This is a happy path since we can just count off items from the
		// end of our existing list until we find one that is not the same
		// as the last.
		var i int
		n := len(l)
		for i = n - 1; i >= 0; i-- {
			if !l[i].Same(l[n-1]) {
				break
			}
		}
		if i < 0 {
			i = 0
		}
		return l[i:]
	}

	// For an unsorted list we'll allocate so that we can construct a new,
	// filtered slice.
	ret := make(List, 0, 1) // one item is the common case, in the absense of build metadata
	example := l.Newest()
	for _, v := range l {
		if v.Same(example) {
			ret = append(ret, v)
		}
	}
	return ret
}

// Set returns a finite Set containing the versions in the receiver.
//
// Although it is possible to recover a list from the return value using
// its List method, the result may be in a different order and will have
// any duplicate elements from the receiving list consolidated.
func (l List) Set() Set {
	return Selection(l...)
}

func (l List) Len() int {
	return len(l)
}

func (l List) Less(i, j int) bool {
	return l[i].LessThan(l[j])
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Sort applies an in-place sort on the list, preserving the relative order of
// any elements that differ only in build metadata. Earlier versions sort
// first, so the newest versions will be at the highest indices in the list
// once this method returns.
func (l List) Sort() {
	sort.Stable(l)
}

// IsSorted returns true if the list is already in ascending order by
// version priority.
func (l List) IsSorted() bool {
	return sort.IsSorted(l)
}
