// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestValidateKMSKey(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		in       string
		expected tfdiags.Diagnostics
	}{
		"kms key id": {
			in: "57ff7a43-341d-46b6-aee3-a450c9de6dc8",
		},
		"kms key arn": {
			in: "arn:aws:kms:us-west-2:111122223333:key/57ff7a43-341d-46b6-aee3-a450c9de6dc8",
		},
		"kms multi-region key id": {
			in: "mrk-f827515944fb43f9b902a09d2c8b554f",
		},
		"kms multi-region key arn": {
			in: "arn:aws:kms:us-west-2:111122223333:key/mrk-a835af0b39c94b86a21a8fc9535df681",
		},
		"kms key alias": {
			in: "alias/arbitrary-key",
		},
		"kms key alias arn": {
			in: "arn:aws:kms:us-west-2:111122223333:alias/arbitrary-key",
		},
		"invalid key": {
			in: "$%wrongkey",
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ID",
					`Value must be a valid KMS Key ID, got "$%wrongkey"`,
					path,
				),
			},
		},
		"non-kms arn": {
			in: "arn:aws:lamda:foo:bar:key/xyz",
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ARN",
					`Value must be a valid KMS Key ARN, got "arn:aws:lamda:foo:bar:key/xyz"`,
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diags := validateKMSKey(path, testcase.in)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateKeyARN(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		in       string
		expected tfdiags.Diagnostics
	}{
		"kms key id": {
			in: "arn:aws:kms:us-west-2:123456789012:key/57ff7a43-341d-46b6-aee3-a450c9de6dc8",
		},
		"kms mrk key id": {
			in: "arn:aws:kms:us-west-2:111122223333:key/mrk-a835af0b39c94b86a21a8fc9535df681",
		},
		"kms non-key id": {
			in: "arn:aws:kms:us-west-2:123456789012:something/else",
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ARN",
					`Value must be a valid KMS Key ARN, got "arn:aws:kms:us-west-2:123456789012:something/else"`,
					path,
				),
			},
		},
		"non-kms arn": {
			in: "arn:aws:iam::123456789012:user/David",
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ARN",
					`Value must be a valid KMS Key ARN, got "arn:aws:iam::123456789012:user/David"`,
					path,
				),
			},
		},
		"not an arn": {
			in: "not an arn",
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ARN",
					`Value must be a valid KMS Key ARN, got "not an arn"`,
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diags := validateKMSKeyARN(path, testcase.in)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateStringLenBetween(t *testing.T) {
	t.Parallel()

	const min, max = 2, 5
	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      string
		expected tfdiags.Diagnostics
	}{
		"valid": {
			val: "valid",
		},

		"too short": {
			val: "x",
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value Length",
					fmt.Sprintf("Length must be between %d and %d, had %d", min, max, 1),
					path,
				),
			},
		},

		"too long": {
			val: "a very long string",
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value Length",
					fmt.Sprintf("Length must be between %d and %d, had %d", min, max, 18),
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateStringLenBetween(min, max)(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateStringMatches(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      string
		re       *regexp.Regexp
		expected tfdiags.Diagnostics
	}{
		"valid": {
			val: "ok",
			re:  regexp.MustCompile(`^o[j-l]?$`),
		},

		"invalid": {
			val: "not ok",
			re:  regexp.MustCompile(`^o[j-l]?$`),
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					"Value must be like ok",
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateStringMatches(testcase.re, "Value must be like ok")(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateARN(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val       string
		validator arnValidator
		expected  tfdiags.Diagnostics
	}{
		"valid": {
			val: "arn:aws:kms:us-west-2:111122223333:key/57ff7a43-341d-46b6-aee3-a450c9de6dc8",
		},

		"invalid": {
			val: "not an ARN",
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid ARN",
					fmt.Sprintf("The value %q cannot be parsed as an ARN: %s", "not an ARN", arnParseError("not an ARN")),
					path,
				),
			},
		},

		"fails validator": {
			val: "arn:aws:kms:us-west-2:111122223333:key/57ff7a43-341d-46b6-aee3-a450c9de6dc8",
			validator: func(val arn.ARN, path cty.Path, diags *tfdiags.Diagnostics) {
				*diags = diags.Append(attributeErrDiag(
					"Test",
					"Test",
					path,
				))
			},
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Test",
					"Test",
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var validators []arnValidator
			if testcase.validator != nil {
				validators = []arnValidator{
					testcase.validator,
				}
			}

			var diags tfdiags.Diagnostics
			validateARN(validators...)(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func arnParseError(s string) error {
	_, err := arn.Parse(s)
	return err
}

func TestValidateIAMPolicyDocument(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      string
		expected tfdiags.Diagnostics
	}{
		"empty object": {
			val: `{}`,
			// Valid JSON, not valid IAM policy (but passes provider's test)
		},
		"array": {
			val: `{"abc":["1","2"]}`,
			// Valid JSON, not valid IAM policy (but passes provider's test)
		},
		"invalid key": {
			val: `{0:"1"}`,
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid JSON Document",
					"The JSON document contains an error: invalid character '0' looking for beginning of object key string, at byte offset 2",
					path,
				),
			},
		},
		"leading whitespace": {
			val: `    {"xyz": "foo"}`,
			// Valid, must be trimmed before passing to AWS
		},
		"is a string": {
			val: `"blub"`,
			// Valid JSON, not valid IAM policy
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid IAM Policy Document",
					`Expected a JSON object describing the policy, had a JSON-encoded string.`,
					path,
				),
			},
		},
		"contains filename": {
			val: `"../some-filename.json"`,
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid IAM Policy Document",
					`Expected a JSON object describing the policy, had a JSON-encoded string.

The string "../some-filename.json" looks like a filename, please pass the contents of the file instead of the filename.`,
					path,
				),
			},
		},
		"double encoded": {
			val: `"{\"Version\":\"...\"}"`,
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid IAM Policy Document",
					`Expected a JSON object describing the policy, had a JSON-encoded string.

The string content was valid JSON, your policy document may have been double-encoded.`,
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateIAMPolicyDocument(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateSetStringElements(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val       cty.Value
		validator stringValidator
		expected  tfdiags.Diagnostics
	}{
		"valid": {
			val: cty.SetVal([]cty.Value{
				cty.StringVal("valid"),
				cty.StringVal("also valid"),
			}),
		},

		"fails validator": {
			val: cty.SetVal([]cty.Value{
				cty.StringVal("valid"),
				cty.StringVal("invalid"),
			}),
			validator: func(val string, path cty.Path, diags *tfdiags.Diagnostics) {
				if val == "invalid" {
					*diags = diags.Append(attributeErrDiag(
						"Test",
						"Test",
						path,
					))
				}
			},
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Test",
					"Test",
					path.Index(cty.StringVal("invalid")),
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var validators []stringValidator
			if testcase.validator != nil {
				validators = []stringValidator{
					testcase.validator,
				}
			}

			var diags tfdiags.Diagnostics
			validateSetStringElements(validators...)(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

// func TestValidateStringSetValues(t *testing.T) {
// 	t.Parallel()

// 	path := cty.GetAttrPath("field")

// 	testcases := map[string]struct {
// 		val       []string
// 		validator stringValidator
// 		expected  tfdiags.Diagnostics
// 	}{
// 		"valid": {
// 			val: []string{
// 				"valid",
// 				"also valid",
// 			},
// 		},

// 		"fails validator": {
// 			val: []string{
// 				"valid",
// 				"invalid",
// 			},
// 			validator: func(val string, path cty.Path, diags *tfdiags.Diagnostics) {
// 				if val == "invalid" {
// 					*diags = diags.Append(attributeErrDiag(
// 						"Test",
// 						"Test",
// 						path,
// 					))
// 				}
// 			},
// 			expected: tfdiags.Diagnostics{
// 				attributeErrDiag(
// 					"Test",
// 					"Test",
// 					path.Index(cty.StringVal("invalid")),
// 				),
// 			},
// 		},
// 	}

// 	for name, testcase := range testcases {
// 		testcase := testcase
// 		t.Run(name, func(t *testing.T) {
// 			t.Parallel()

// 			var validators []stringValidator
// 			if testcase.validator != nil {
// 				validators = []stringValidator{
// 					testcase.validator,
// 				}
// 			}

// 			var diags tfdiags.Diagnostics
// 			validateStringSetValues(validators...)(testcase.val, path, &diags)

// 			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
// 				t.Errorf("unexpected diagnostics difference: %s", diff)
// 			}
// 		})
// 	}
// }

func TestValidateDuration(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val       string
		validator durationValidator
		expected  tfdiags.Diagnostics
	}{
		"valid": {
			val: "1h",
		},

		"invalid": {
			val: "one hour",
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Duration",
					fmt.Sprintf("The value %q cannot be parsed as a duration: %s", "one hour", durationParseError("one hour")),
					path,
				),
			},
		},

		"fails validator": {
			val: "1h",
			validator: func(val time.Duration, path cty.Path, diags *tfdiags.Diagnostics) {
				*diags = diags.Append(attributeErrDiag(
					"Test",
					"Test",
					path,
				))
			},
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Test",
					"Test",
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var validators []durationValidator
			if testcase.validator != nil {
				validators = []durationValidator{
					testcase.validator,
				}
			}

			var diags tfdiags.Diagnostics
			validateDuration(validators...)(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func durationParseError(s string) error {
	_, err := time.ParseDuration(s)
	return err
}

func TestValidateDurationBetween(t *testing.T) {
	t.Parallel()

	const min, max = 15 * time.Minute, 12 * time.Hour
	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      time.Duration
		expected tfdiags.Diagnostics
	}{
		"valid": {
			val: 1 * time.Hour,
		},

		"too short": {
			val: 1 * time.Minute,
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Duration",
					fmt.Sprintf("Duration must be between %s and %s, had %s", min, max, 1*time.Minute),
					path,
				),
			},
		},

		"too long": {
			val: 24 * time.Hour,
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Duration",
					fmt.Sprintf("Duration must be between %s and %s, had %s", min, max, 24*time.Hour),
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateDurationBetween(min, max)(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateStringLegacyURL(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      string
		expected tfdiags.Diagnostics
	}{
		"no trailing slash": {
			val: "https://domain.test",
		},

		"no path": {
			val: "https://domain.test/",
		},

		"with path": {
			val: "https://domain.test/path",
		},

		"with port no trailing slash": {
			val: "https://domain.test:1234",
		},

		"with port no path": {
			val: "https://domain.test:1234/",
		},

		"with port with path": {
			val: "https://domain.test:1234/path",
		},

		"no scheme no trailing slash": {
			val: "domain.test",
			expected: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("domain.test", path),
			},
		},

		"no scheme no path": {
			val: "domain.test/",
			expected: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("domain.test/", path),
			},
		},

		"no scheme with path": {
			val: "domain.test/path",
			expected: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("domain.test/path", path),
			},
		},

		"no scheme with port": {
			val: "domain.test:1234",
			expected: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("domain.test:1234", path),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateStringLegacyURL(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateStringValidURL(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      string
		expected tfdiags.Diagnostics
	}{
		"no trailing slash": {
			val: "https://domain.test",
		},

		"no path": {
			val: "https://domain.test/",
		},

		"with path": {
			val: "https://domain.test/path",
		},

		"with port no trailing slash": {
			val: "https://domain.test:1234",
		},

		"with port no path": {
			val: "https://domain.test:1234/",
		},

		"with port with path": {
			val: "https://domain.test:1234/path",
		},

		"no scheme no trailing slash": {
			val: "domain.test",
			expected: tfdiags.Diagnostics{
				invalidURLDiag("domain.test", path),
			},
		},

		"no scheme no path": {
			val: "domain.test/",
			expected: tfdiags.Diagnostics{
				invalidURLDiag("domain.test/", path),
			},
		},

		"no scheme with path": {
			val: "domain.test/path",
			expected: tfdiags.Diagnostics{
				invalidURLDiag("domain.test/path", path),
			},
		},

		"no scheme with port": {
			val: "domain.test:1234",
			expected: tfdiags.Diagnostics{
				invalidURLDiag("domain.test:1234", path),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateStringValidURL(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func Test_validateStringDoesNotContain(t *testing.T) {
	t.Parallel()

	path := cty.GetAttrPath("field")

	testcases := map[string]struct {
		val      string
		s        string
		expected tfdiags.Diagnostics
	}{
		"valid": {
			val: "foo",
			s:   "bar",
		},

		"invalid": {
			val: "foobarbaz",
			s:   "bar",
			expected: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`Value must not contain "bar"`,
					path,
				),
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags tfdiags.Diagnostics
			validateStringDoesNotContain(testcase.s)(testcase.val, path, &diags)

			if diff := cmp.Diff(diags, testcase.expected, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}
