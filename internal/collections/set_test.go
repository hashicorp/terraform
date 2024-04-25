// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import (
	"testing"
)

func TestSet(t *testing.T) {
	set := NewSet[testingKey]()

	if got, want := set.Len(), 0; got != want {
		t.Errorf("wrong initial number of elements\ngot:  %#v\nwant: %#v", got, want)
	}

	set.Add(testingKey("a"))
	if got, want := set.Len(), 1; got != want {
		t.Errorf("wrong number of elements after adding \"a\"\ngot:  %#v\nwant: %#v", got, want)
	}

	set.Add(testingKey("a"))
	set.Add(testingKey("b"))
	if got, want := set.Len(), 2; got != want {
		t.Errorf("wrong number of elements after re-adding \"a\" and adding \"b\"\ngot:  %#v\nwant: %#v", got, want)
	}

	set.Remove(testingKey("a"))
	if got, want := set.Len(), 1; got != want {
		t.Errorf("wrong number of elements after removing \"a\"\ngot:  %#v\nwant: %#v", got, want)
	}

	if got, want := set.Has(testingKey("a")), false; got != want {
		t.Errorf("set still has \"a\" after removing it")
	}
	if got, want := set.Has(testingKey("b")), true; got != want {
		t.Errorf("set doesn't have \"b\" after adding it")
	}
}

func TestSetUninit(t *testing.T) {
	// An zero-value set should behave like it's empty for read-only operations.
	var zeroSet Set[string]
	if got, want := zeroSet.Len(), 0; got != want {
		t.Errorf("wrong number of elements\ngot:  %d\nwant: %d", got, want)
	}
	if zeroSet.Has("anything") {
		// (this is really just testing that we can call Has without panicking;
		// it's unlikely that this would ever fail by successfully lying about
		// a particular member being present.)
		t.Error("Has reported that \"anything\" is present")
	}
}
