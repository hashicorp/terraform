package newrelic

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

func TestValidationIntInInSlice(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 2,
			f:   intInSlice([]int{1, 2, 3}),
		},
		{
			val:         4,
			f:           intInSlice([]int{1, 2, 3}),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be one of \\[1 2 3\\], got 4"),
		},
		{
			val:         "foo",
			f:           intInSlice([]int{1, 2, 3}),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be int"),
		},
	})
}

func TestValidationFloat64Gte(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 1.1,
			f:   float64Gte(1.1),
		},
		{
			val: 1.2,
			f:   float64Gte(1.1),
		},
		{
			val:         "foo",
			f:           float64Gte(1.1),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be float64"),
		},
		{
			val:         0.1,
			f:           float64Gte(1.1),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be greater than or equal to 1.1, got 0.1"),
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
