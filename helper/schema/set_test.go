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
