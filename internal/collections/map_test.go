// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import "testing"

func TestMap(t *testing.T) {
	m := NewMap[testingKey, string]()

	if got, want := m.Len(), 0; got != want {
		t.Errorf("wrong initial number of elements\ngot:  %#v\nwant: %#v", got, want)
	}

	m.Put(testingKey("a"), "A")
	if got, want := m.Len(), 1; got != want {
		t.Errorf("wrong number of elements after adding \"a\"\ngot:  %#v\nwant: %#v", got, want)
	}
	if got, want := m.Get(testingKey("a")), "A"; got != want {
		t.Errorf("wrong value for \"a\"\ngot:  %#v\nwant: %#v", got, want)
	}

	m.Put(testingKey("a"), "A'")
	if got, want := m.Len(), 1; got != want {
		t.Errorf("wrong number of elements after re-adding \"a\"\ngot:  %#v\nwant: %#v", got, want)
	}
	if got, want := m.Get(testingKey("a")), "A'"; got != want {
		t.Errorf("wrong updated value for \"a\"\ngot:  %#v\nwant: %#v", got, want)
	}

	m.Delete(testingKey("a"))
	if got, want := m.Len(), 0; got != want {
		t.Errorf("wrong number of elements after removing \"m\"\ngot:  %#v\nwant: %#v", got, want)
	}
}
