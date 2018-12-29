package structure

import (
	"testing"
)

func TestFlattenJson_empty(t *testing.T) {
	input := make(map[string]interface{}, 0)
	expected := ""
	actual, err := FlattenJsonToString(input)
	if err != nil {
		t.Fatalf("Expected not to throw an error while Flattening JSON, but got: %s", err)
	}

	if expected != actual {
		t.Fatalf("Got: `%+v`. Expected: `%+v`", actual, expected)
	}
}

func TestFlattenJson_singleItem(t *testing.T) {
	input := make(map[string]interface{}, 1)
	input["foo"] = "bar"
	expected := `{"foo":"bar"}`
	actual, err := FlattenJsonToString(input)
	if err != nil {
		t.Fatalf("Expected not to throw an error while Flattening JSON, but got: %s", err)
	}

	if expected != actual {
		t.Fatalf("Got: `%+v`. Expected: `%+v`", actual, expected)
	}
}

func TestFlattenJson_multipleItems(t *testing.T) {
	input := make(map[string]interface{}, 1)
	input["foo"] = "bar"
	input["bar"] = "foo"
	expected := `{"bar":"foo","foo":"bar"}`
	actual, err := FlattenJsonToString(input)
	if err != nil {
		t.Fatalf("Expected not to throw an error while Flattening JSON, but got: %s", err)
	}

	if expected != actual {
		t.Fatalf("Got: `%+v`. Expected: `%+v`", actual, expected)
	}
}
