package structure

import (
	"testing"
)

func TestNormalizeJsonString_valid(t *testing.T) {
	// Well formatted and valid.
	validJson := `{
   "abc": {
      "def": 123,
      "xyz": [
         {
            "a": "ホリネズミ"
         },
         {
            "b": "1\\n2"
         }
      ]
   }
}`
	expected := `{"abc":{"def":123,"xyz":[{"a":"ホリネズミ"},{"b":"1\\n2"}]}}`

	actual, err := NormalizeJsonString(validJson)
	if err != nil {
		t.Fatalf("Expected not to throw an error while parsing JSON, but got: %s", err)
	}

	if actual != expected {
		t.Fatalf("Got:\n\n%s\n\nExpected:\n\n%s\n", actual, expected)
	}

	// Well formatted but not valid,
	// missing closing square bracket.
	invalidJson := `{
   "abc": {
      "def": 123,
      "xyz": [
         {
            "a": "1"
         }
      }
   }
}`
	actual, err = NormalizeJsonString(invalidJson)
	if err == nil {
		t.Fatalf("Expected to throw an error while parsing JSON, but got: %s", err)
	}

	// We expect the invalid JSON to be shown back to us again.
	if actual != invalidJson {
		t.Fatalf("Got:\n\n%s\n\nExpected:\n\n%s\n", actual, invalidJson)
	}

	// Verify that it leaves strings alone
	testString := "2016-07-28t04:07:02z\nsomething else"
	expected = "2016-07-28t04:07:02z\nsomething else"
	actual, err = NormalizeJsonString(testString)
	if err == nil {
		t.Fatalf("Expected to throw an error while parsing JSON, but got: %s", err)
	}

	if actual != expected {
		t.Fatalf("Got:\n\n%s\n\nExpected:\n\n%s\n", actual, expected)
	}
}

func TestNormalizeJsonString_invalid(t *testing.T) {
	// Well formatted but not valid,
	// missing closing squre bracket.
	invalidJson := `{
   "abc": {
      "def": 123,
      "xyz": [
         {
            "a": "1"
         }
      }
   }
}`
	expected := `{"abc":{"def":123,"xyz":[{"a":"ホリネズミ"},{"b":"1\\n2"}]}}`
	actual, err := NormalizeJsonString(invalidJson)
	if err == nil {
		t.Fatalf("Expected to throw an error while parsing JSON, but got: %s", err)
	}

	// We expect the invalid JSON to be shown back to us again.
	if actual != invalidJson {
		t.Fatalf("Got:\n\n%s\n\nExpected:\n\n%s\n", expected, invalidJson)
	}
}
