package akamai

import (
    "encoding/json"
    "reflect"
    "testing"
)

func TestNewAkamaiError(t *testing.T) {
    bodyContents := []byte(`{
  "type" : "https://problems.luna.akamaiapis.net/papi/v0/http/not-found",
  "title" : "Not Found",
  "detail" : "The system was unable to locate the requested resource.",
  "status" : 404,
  "instance" : "https://private-anon-960581d538-akamaiopen2lunapapiproduction.apiary-mock.com/papi/v0/group#12c03c3b-016a-4545-96b1-5cc0b84cde49"
}`)

    actual, err := NewAkamaiError(bodyContents)
    if err != nil {
        t.Fatalf("err: %s", err)
    }

    expected := &AkamaiError{
        Type:   "https://problems.luna.akamaiapis.net/papi/v0/http/not-found",
        Title:  "Not Found",
        Detail: "The system was unable to locate the requested resource.",
        Status: 404,
    }

    if !reflect.DeepEqual(actual, expected) {
        t.Fatalf("bad: %#v", actual)
    }
}

func TestNewAkamaiError_InvalidJSON(t *testing.T) {
    bodyContents := []byte(`{`)

    _, err := NewAkamaiError(bodyContents)

    if err == nil {
        t.Fatalf("Expected error of type %T", json.SyntaxError{})
    }
}

func TestError(t *testing.T) {
    akamaiError := &AkamaiError{
        Type:   "https://problems.luna.akamaiapis.net/papi/v0/http/not-found",
        Title:  "Not Found",
        Detail: "The system was unable to locate the requested resource.",
        Status: 404,
    }

    actual := akamaiError.Error()
    expected := "404 Not Found\nThe system was unable to locate the requested resource."

    if actual != expected {
        t.Fatalf("Expected error string to be %q, got %q", expected, actual)
    }
}
