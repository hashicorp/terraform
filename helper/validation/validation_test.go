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

func TestValidationSringInSlice(t *testing.T) {
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
