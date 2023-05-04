// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"reflect"
	"testing"
)

func TestSetAdd(t *testing.T) {
	s := &Set{F: testSetInt}
	s.Add(1)
	s.Add(5)
	s.Add(25)

	expected := []interface{}{1, 25, 5}
	actual := s.List()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestSetAdd_negative(t *testing.T) {
	// Since we don't allow negative hashes, this should just hash to the
	// same thing...
	s := &Set{F: testSetInt}
	s.Add(-1)
	s.Add(1)

	expected := []interface{}{-1}
	actual := s.List()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestSetContains(t *testing.T) {
	s := &Set{F: testSetInt}
	s.Add(5)
	s.Add(-5)

	if s.Contains(2) {
		t.Fatal("should not contain")
	}
	if !s.Contains(5) {
		t.Fatal("should contain")
	}
	if !s.Contains(-5) {
		t.Fatal("should contain")
	}
}

func TestSetDifference(t *testing.T) {
	s1 := &Set{F: testSetInt}
	s2 := &Set{F: testSetInt}

	s1.Add(1)
	s1.Add(5)

	s2.Add(5)
	s2.Add(25)

	difference := s1.Difference(s2)
	difference.Add(2)

	expected := []interface{}{1, 2}
	actual := difference.List()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestSetIntersection(t *testing.T) {
	s1 := &Set{F: testSetInt}
	s2 := &Set{F: testSetInt}

	s1.Add(1)
	s1.Add(5)

	s2.Add(5)
	s2.Add(25)

	intersection := s1.Intersection(s2)
	intersection.Add(2)

	expected := []interface{}{2, 5}
	actual := intersection.List()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestSetUnion(t *testing.T) {
	s1 := &Set{F: testSetInt}
	s2 := &Set{F: testSetInt}

	s1.Add(1)
	s1.Add(5)

	s2.Add(5)
	s2.Add(25)

	union := s1.Union(s2)
	union.Add(2)

	expected := []interface{}{1, 2, 25, 5}
	actual := union.List()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func testSetInt(v interface{}) int {
	return v.(int)
}

func TestHashResource_nil(t *testing.T) {
	resource := &Resource{
		Schema: map[string]*Schema{
			"name": {
				Type:     TypeString,
				Optional: true,
			},
		},
	}
	f := HashResource(resource)

	idx := f(nil)
	if idx != 0 {
		t.Fatalf("Expected 0 when hashing nil, given: %d", idx)
	}
}

func TestHashEqual(t *testing.T) {
	nested := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeString,
				Optional: true,
			},
		},
	}
	root := &Resource{
		Schema: map[string]*Schema{
			"bar": {
				Type:     TypeString,
				Optional: true,
			},
			"nested": {
				Type:     TypeSet,
				Optional: true,
				Elem:     nested,
			},
		},
	}
	n1 := map[string]interface{}{"foo": "bar"}
	n2 := map[string]interface{}{"foo": "baz"}

	r1 := map[string]interface{}{
		"bar":    "baz",
		"nested": NewSet(HashResource(nested), []interface{}{n1}),
	}
	r2 := map[string]interface{}{
		"bar":    "qux",
		"nested": NewSet(HashResource(nested), []interface{}{n2}),
	}
	r3 := map[string]interface{}{
		"bar":    "baz",
		"nested": NewSet(HashResource(nested), []interface{}{n2}),
	}
	r4 := map[string]interface{}{
		"bar":    "qux",
		"nested": NewSet(HashResource(nested), []interface{}{n1}),
	}
	s1 := NewSet(HashResource(root), []interface{}{r1})
	s2 := NewSet(HashResource(root), []interface{}{r2})
	s3 := NewSet(HashResource(root), []interface{}{r3})
	s4 := NewSet(HashResource(root), []interface{}{r4})

	cases := []struct {
		name     string
		set      *Set
		compare  *Set
		expected bool
	}{
		{
			name:     "equal",
			set:      s1,
			compare:  s1,
			expected: true,
		},
		{
			name:     "not equal",
			set:      s1,
			compare:  s2,
			expected: false,
		},
		{
			name:     "outer equal, should still not be equal",
			set:      s1,
			compare:  s3,
			expected: false,
		},
		{
			name:     "inner equal, should still not be equal",
			set:      s1,
			compare:  s4,
			expected: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.set.HashEqual(tc.compare)
			if tc.expected != actual {
				t.Fatalf("expected %t, got %t", tc.expected, actual)
			}
		})
	}
}
