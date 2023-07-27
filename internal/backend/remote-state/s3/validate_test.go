package s3

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestValidateKMSKey(t *testing.T) {
	t.Parallel()

	path := cty.Path{cty.GetAttrStep{Name: "field"}}

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
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ID",
					`Value must be a valid KMS Key ID, got "alias/arbitrary-key"`,
					path,
				),
			},
		},
		"kms key alias arn": {
			in: "arn:aws:kms:us-west-2:111122223333:alias/arbitrary-key",
			expected: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid KMS Key ARN",
					`Value must be a valid KMS Key ARN, got "arn:aws:kms:us-west-2:111122223333:alias/arbitrary-key"`,
					path,
				),
			},
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

			if diff := cmp.Diff(diags, testcase.expected, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateKeyARN(t *testing.T) {
	t.Parallel()

	path := cty.Path{cty.GetAttrStep{Name: "field"}}

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

			if diff := cmp.Diff(diags, testcase.expected, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateStringLenBetween(t *testing.T) {
	t.Parallel()

	const min, max = 2, 5
	path := cty.Path{cty.GetAttrStep{Name: "field"}}

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

			if diff := cmp.Diff(diags, testcase.expected, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestValidateStringMatches(t *testing.T) {
	t.Parallel()

	path := cty.Path{cty.GetAttrStep{Name: "field"}}

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

			if diff := cmp.Diff(diags, testcase.expected, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}
