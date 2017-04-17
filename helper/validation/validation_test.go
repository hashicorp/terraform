package validation

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

type testCase struct {
	val         interface{}
	f           schema.SchemaValidateFunc
	expectedErr *regexp.Regexp
}

func TestValidationIntBetween(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 1,
			f:   IntBetween(1, 1),
		},
		{
			val: 1,
			f:   IntBetween(0, 2),
		},
		{
			val:         1,
			f:           IntBetween(2, 3),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be in the range \\(2 - 3\\), got 1"),
		},
		{
			val:         "1",
			f:           IntBetween(2, 3),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be int"),
		},
	})
}

func TestValidationStringInSlice(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "ValidValue",
			f:   StringInSlice([]string{"ValidValue", "AnotherValidValue"}, false),
		},
		// ignore case
		{
			val: "VALIDVALUE",
			f:   StringInSlice([]string{"ValidValue", "AnotherValidValue"}, true),
		},
		{
			val:         "VALIDVALUE",
			f:           StringInSlice([]string{"ValidValue", "AnotherValidValue"}, false),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be one of \\[ValidValue AnotherValidValue\\], got VALIDVALUE"),
		},
		{
			val:         "InvalidValue",
			f:           StringInSlice([]string{"ValidValue", "AnotherValidValue"}, false),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be one of \\[ValidValue AnotherValidValue\\], got InvalidValue"),
		},
		{
			val:         1,
			f:           StringInSlice([]string{"ValidValue", "AnotherValidValue"}, false),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be string"),
		},
	})
}

func TestValidateJsonString(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    `{0:"1"}`,
			ErrCount: 1,
		},
		{
			Value:    `{'abc':1}`,
			ErrCount: 1,
		},
		{
			Value:    `{"def":}`,
			ErrCount: 1,
		},
		{
			Value:    `{"xyz":[}}`,
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := ValidateJsonString(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    ``,
			ErrCount: 0,
		},
		{
			Value:    `{}`,
			ErrCount: 0,
		},
		{
			Value:    `{"abc":["1","2"]}`,
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := ValidateJsonString(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func runTestCases(t *testing.T, cases []testCase) {
	matchErr := func(errs []error, r *regexp.Regexp) bool {
		// err must match one provided
		for _, err := range errs {
			if r.MatchString(err.Error()) {
				return true
			}
		}

		return false
	}

	for i, tc := range cases {
		_, errs := tc.f(tc.val, "test_property")

		if len(errs) == 0 && tc.expectedErr == nil {
			continue
		}

		if !matchErr(errs, tc.expectedErr) {
			t.Fatalf("expected test case %d to produce error matching \"%s\", got %v", i, tc.expectedErr, errs)
		}
	}
}
