package config

import (
	"reflect"
	"testing"
)

func TestStringList_slice(t *testing.T) {
	expected := []string{"apple", "banana", "pear"}
	l := NewStringList(expected)
	actual := l.Slice()

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %q, got %q", expected, actual)
	}
}

func TestStringList_element(t *testing.T) {
	list := []string{"apple", "banana", "pear"}
	l := NewStringList(list)
	actual := l.Element(1)

	expected := "banana"

	if actual != expected {
		t.Fatalf("Expected 2nd element from %q to be %q, got %q",
			list, expected, actual)
	}
}

func TestStringList_empty_slice(t *testing.T) {
	expected := []string{}
	l := NewStringList(expected)
	actual := l.Slice()

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %q, got %q", expected, actual)
	}
}

func TestStringList_empty_slice_length(t *testing.T) {
	list := []string{}
	l := NewStringList([]string{})
	actual := l.Length()

	expected := 0

	if actual != expected {
		t.Fatalf("Expected length of %q to be %d, got %d",
			list, expected, actual)
	}
}
