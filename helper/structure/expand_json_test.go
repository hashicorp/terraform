package structure

import (
	"reflect"
	"testing"
)

func TestExpandJson_emptyString(t *testing.T) {
	_, err := ExpandJsonFromString("")
	if err == nil {
		t.Fatal("Expected to throw an error while Expanding JSON")
	}
}

func TestExpandJson_singleItem(t *testing.T) {
	input := `{
	  "foo": "bar"
	}`
	expected := make(map[string]interface{}, 1)
	expected["foo"] = "bar"
	actual, err := ExpandJsonFromString(input)
	if err != nil {
		t.Fatalf("Expected not to throw an error while Expanding JSON, but got: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Got:\n\n%+v\n\nExpected:\n\n%+v\n", actual, expected)
	}
}

func TestExpandJson_multipleItems(t *testing.T) {
	input := `{
	  "foo": "bar",
	  "hello": "world"
	}`
	expected := make(map[string]interface{}, 1)
	expected["foo"] = "bar"
	expected["hello"] = "world"

	actual, err := ExpandJsonFromString(input)
	if err != nil {
		t.Fatalf("Expected not to throw an error while Expanding JSON, but got: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Got:\n\n%+v\n\nExpected:\n\n%+v\n", actual, expected)
	}
}
