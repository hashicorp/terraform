package azurerm

import "testing"

func TestNormalizeJsonString(t *testing.T) {
	var err error
	var actual string

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

	actual, err = normalizeJsonString(validJson)
	if err != nil {
		t.Fatalf("Expected not to throw an error while parsing JSON, but got: %s", err)
	}

	if actual != expected {
		t.Fatalf("Got:\n\n%s\n\nExpected:\n\n%s\n", actual, expected)
	}

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
	actual, err = normalizeJsonString(invalidJson)
	if err == nil {
		t.Fatalf("Expected to throw an error while parsing JSON, but got: %s", err)
	}

	// We expect the invalid JSON to be shown back to us again.
	if actual != invalidJson {
		t.Fatalf("Got:\n\n%s\n\nExpected:\n\n%s\n", expected, invalidJson)
	}
}
