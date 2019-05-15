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

func TestValidationAll(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "valid",
			f: All(
				StringLenBetween(5, 42),
				StringMatch(regexp.MustCompile(`[a-zA-Z0-9]+`), "value must be alphanumeric"),
			),
		},
		{
			val: "foo",
			f: All(
				StringLenBetween(5, 42),
				StringMatch(regexp.MustCompile(`[a-zA-Z0-9]+`), "value must be alphanumeric"),
			),
			expectedErr: regexp.MustCompile("expected length of [\\w]+ to be in the range \\(5 - 42\\), got foo"),
		},
		{
			val: "!!!!!",
			f: All(
				StringLenBetween(5, 42),
				StringMatch(regexp.MustCompile(`[a-zA-Z0-9]+`), "value must be alphanumeric"),
			),
			expectedErr: regexp.MustCompile("value must be alphanumeric"),
		},
	})
}

func TestValidationAny(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 43,
			f: Any(
				IntAtLeast(42),
				IntAtMost(5),
			),
		},
		{
			val: 4,
			f: Any(
				IntAtLeast(42),
				IntAtMost(5),
			),
		},
		{
			val: 7,
			f: Any(
				IntAtLeast(42),
				IntAtMost(5),
			),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be at least \\(42\\), got 7"),
		},
		{
			val: 7,
			f: Any(
				IntAtLeast(42),
				IntAtMost(5),
			),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be at most \\(5\\), got 7"),
		},
	})
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

func TestValidationIntAtLeast(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 1,
			f:   IntAtLeast(1),
		},
		{
			val: 1,
			f:   IntAtLeast(0),
		},
		{
			val:         1,
			f:           IntAtLeast(2),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be at least \\(2\\), got 1"),
		},
		{
			val:         "1",
			f:           IntAtLeast(2),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be int"),
		},
	})
}

func TestValidationIntAtMost(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 1,
			f:   IntAtMost(1),
		},
		{
			val: 1,
			f:   IntAtMost(2),
		},
		{
			val:         1,
			f:           IntAtMost(0),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be at most \\(0\\), got 1"),
		},
		{
			val:         "1",
			f:           IntAtMost(0),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be int"),
		},
	})
}

func TestValidationIntInSlice(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: 42,
			f:   IntInSlice([]int{1, 42}),
		},
		{
			val:         42,
			f:           IntInSlice([]int{10, 20}),
			expectedErr: regexp.MustCompile("expected [\\w]+ to be one of \\[10 20\\], got 42"),
		},
		{
			val:         "InvalidValue",
			f:           IntInSlice([]int{10, 20}),
			expectedErr: regexp.MustCompile("expected type of [\\w]+ to be an integer"),
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

func TestValidationStringMatch(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "foobar",
			f:   StringMatch(regexp.MustCompile(".*foo.*"), ""),
		},
		{
			val:         "bar",
			f:           StringMatch(regexp.MustCompile(".*foo.*"), ""),
			expectedErr: regexp.MustCompile("expected value of [\\w]+ to match regular expression " + regexp.QuoteMeta(`".*foo.*"`)),
		},
		{
			val:         "bar",
			f:           StringMatch(regexp.MustCompile(".*foo.*"), "value must contain foo"),
			expectedErr: regexp.MustCompile("invalid value for [\\w]+ \\(value must contain foo\\)"),
		},
	})
}

func TestValidationRegexp(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: ".*foo.*",
			f:   ValidateRegexp,
		},
		{
			val:         "foo(bar",
			f:           ValidateRegexp,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta("error parsing regexp: missing closing ): `foo(bar`")),
		},
	})
}

func TestValidationSingleIP(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "172.10.10.10",
			f:   SingleIP(),
		},
		{
			val:         "1.1.1",
			f:           SingleIP(),
			expectedErr: regexp.MustCompile(regexp.QuoteMeta("expected test_property to contain a valid IP, got:")),
		},
		{
			val:         "1.1.1.0/20",
			f:           SingleIP(),
			expectedErr: regexp.MustCompile(regexp.QuoteMeta("expected test_property to contain a valid IP, got:")),
		},
		{
			val:         "256.1.1.1",
			f:           SingleIP(),
			expectedErr: regexp.MustCompile(regexp.QuoteMeta("expected test_property to contain a valid IP, got:")),
		},
	})
}

func TestValidationIPRange(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "172.10.10.10-172.10.10.12",
			f:   IPRange(),
		},
		{
			val:         "172.10.10.20",
			f:           IPRange(),
			expectedErr: regexp.MustCompile(regexp.QuoteMeta("expected test_property to contain a valid IP range, got:")),
		},
		{
			val:         "172.10.10.20-172.10.10.12",
			f:           IPRange(),
			expectedErr: regexp.MustCompile(regexp.QuoteMeta("expected test_property to contain a valid IP range, got:")),
		},
	})
}

func TestValidateRFC3339TimeString(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "2018-03-01T00:00:00Z",
			f:   ValidateRFC3339TimeString,
		},
		{
			val: "2018-03-01T00:00:00-05:00",
			f:   ValidateRFC3339TimeString,
		},
		{
			val: "2018-03-01T00:00:00+05:00",
			f:   ValidateRFC3339TimeString,
		},
		{
			val:         "03/01/2018",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
		},
		{
			val:         "03-01-2018",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
		},
		{
			val:         "2018-03-01",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
		},
		{
			val:         "2018-03-01T",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
		},
		{
			val:         "2018-03-01T00:00:00",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
		},
		{
			val:         "2018-03-01T00:00:00Z05:00",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
		},
		{
			val:         "2018-03-01T00:00:00Z-05:00",
			f:           ValidateRFC3339TimeString,
			expectedErr: regexp.MustCompile(regexp.QuoteMeta(`invalid RFC3339 timestamp`)),
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

func TestValidateListUniqueStrings(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: []interface{}{"foo", "bar"},
			f:   ValidateListUniqueStrings,
		},
		{
			val:         []interface{}{"foo", "bar", "foo"},
			f:           ValidateListUniqueStrings,
			expectedErr: regexp.MustCompile("duplicate entry - foo"),
		},
		{
			val:         []interface{}{"foo", "bar", "foo", "baz", "bar"},
			f:           ValidateListUniqueStrings,
			expectedErr: regexp.MustCompile("duplicate entry - (?:foo|bar)"),
		},
	})
}

func TestValidationNoZeroValues(t *testing.T) {
	runTestCases(t, []testCase{
		{
			val: "foo",
			f:   NoZeroValues,
		},
		{
			val: 1,
			f:   NoZeroValues,
		},
		{
			val: float64(1),
			f:   NoZeroValues,
		},
		{
			val:         "",
			f:           NoZeroValues,
			expectedErr: regexp.MustCompile("must not be empty"),
		},
		{
			val:         0,
			f:           NoZeroValues,
			expectedErr: regexp.MustCompile("must not be zero"),
		},
		{
			val:         float64(0),
			f:           NoZeroValues,
			expectedErr: regexp.MustCompile("must not be zero"),
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

		if len(errs) != 0 && tc.expectedErr == nil {
			t.Fatalf("expected test case %d to produce no errors, got %v", i, errs)
		}

		if !matchErr(errs, tc.expectedErr) {
			t.Fatalf("expected test case %d to produce error matching \"%s\", got %v", i, tc.expectedErr, errs)
		}
	}
}

func TestFloatBetween(t *testing.T) {
	cases := map[string]struct {
		Value                  interface{}
		ValidateFunc           schema.SchemaValidateFunc
		ExpectValidationErrors bool
	}{
		"accept valid value": {
			Value:                  1.5,
			ValidateFunc:           FloatBetween(1.0, 2.0),
			ExpectValidationErrors: false,
		},
		"accept valid value inclusive upper bound": {
			Value:                  1.0,
			ValidateFunc:           FloatBetween(0.0, 1.0),
			ExpectValidationErrors: false,
		},
		"accept valid value inclusive lower bound": {
			Value:                  0.0,
			ValidateFunc:           FloatBetween(0.0, 1.0),
			ExpectValidationErrors: false,
		},
		"reject out of range value": {
			Value:                  -1.0,
			ValidateFunc:           FloatBetween(0.0, 1.0),
			ExpectValidationErrors: true,
		},
		"reject incorrectly typed value": {
			Value:                  1,
			ValidateFunc:           FloatBetween(0.0, 1.0),
			ExpectValidationErrors: true,
		},
	}

	for tn, tc := range cases {
		_, errors := tc.ValidateFunc(tc.Value, tn)
		if len(errors) > 0 && !tc.ExpectValidationErrors {
			t.Errorf("%s: unexpected errors %s", tn, errors)
		} else if len(errors) == 0 && tc.ExpectValidationErrors {
			t.Errorf("%s: expected errors but got none", tn)
		}
	}
}
