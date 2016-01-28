package vault

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidateDurationString(t *testing.T) {
	cases := map[string]struct {
		Input          string
		ExpectedErrors []error
	}{
		"simple valid": {
			Input: "10m0s",
		},
		"unparseable": {
			Input: "100nopes",
			ExpectedErrors: []error{errors.New(
				"k: error parsing as duration: time: unknown unit nopes in duration 100nopes")},
		},
	}
	for tn, tc := range cases {
		_, es := ValidateDurationString(tc.Input, "k")
		if !reflect.DeepEqual(es, tc.ExpectedErrors) {
			t.Fatalf("%q: expected errors: %v, got: %v", tn, tc.ExpectedErrors, es)
		}
	}
}

func TestNormalizeDurationString(t *testing.T) {
	cases := map[string]struct {
		Input          string
		ExpectedOutput string
	}{
		"simple": {
			Input:          "10m",
			ExpectedOutput: "10m0s",
		},
		"mins to hours": {
			Input:          "100m1s",
			ExpectedOutput: "1h40m1s",
		},
	}
	for tn, tc := range cases {
		out := NormalizeDurationString(tc.Input)
		if out != tc.ExpectedOutput {
			t.Fatalf("%q: expected: %q, got: %q", tn, tc.ExpectedOutput, out)
		}
	}
}
