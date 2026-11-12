// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

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
