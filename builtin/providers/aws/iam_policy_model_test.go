package aws

import (
	"testing"
)

func TestNormalizeIAMPolicyJSON(t *testing.T) {
	type testCase struct {
		Input       string
		Expected    string
		Normalizer  IAMPolicyStatementNormalizer
		ExpectError bool
	}

	tests := []testCase{
		{
			`{}`,
			`{}`,
			nil,
			false,
		},
		{
			`{"Statement":[]}`,
			`{}`,
			nil,
			false,
		},
		{
			// Single-item action set becomes single string
			`{"Statement":[{"Action":["foo:Baz"]}]}`,
			`{"Statement":[{"Action":"foo:Baz"}]}`,
			nil,
			false,
		},
		{
			// Multiple actions are sorted
			`{"Statement":[{"Action":["foo:Zeek","foo:Baz"]}]}`,
			`{"Statement":[{"Action":["foo:Baz","foo:Zeek"]}]}`,
			nil,
			false,
		},
		{
			`{"Statement":[{"NotAction":["foo:Zeek"]}]}`,
			`{"Statement":[{"NotAction":"foo:Zeek"}]}`,
			nil,
			false,
		},
		{
			`{"Statement":[{"Resource":["foo:Zeek"]}]}`,
			`{"Statement":[{"Resource":"foo:Zeek"}]}`,
			nil,
			false,
		},
		{
			`{"Statement":[{"NotResource":["foo:Zeek"]}]}`,
			`{"Statement":[{"NotResource":"foo:Zeek"}]}`,
			nil,
			false,
		},
		{
			`{"Statement":[{"Principal":{"AWS":["12345"]}}]}`,
			`{"Statement":[{"Principal":{"AWS":"12345"}}]}`,
			nil,
			false,
		},
		{
			`{"Statement":[{"NotPrincipal":{"AWS":["12345"]}}]}`,
			`{"Statement":[{"NotPrincipal":{"AWS":"12345"}}]}`,
			nil,
			false,
		},
		{
			`{"Statement":[{"Condition":{"DataGreaterThan":{"aws:CurrentTime":["abc123"]}}}]}`,
			`{"Statement":[{"Condition":{"DataGreaterThan":{"aws:CurrentTime":"abc123"}}}]}`,
			nil,
			false,
		},
		{
			// Statement attribute order is normalized
			`{"Statement":[{"NotAction":"foo:Zeek","Action":"foo:Baz"}]}`,
			`{"Statement":[{"Action":"foo:Baz","NotAction":"foo:Zeek"}]}`,
			nil,
			false,
		},
		{
			// Unrecognized attributes are discarded
			`{"Statement":[{"Baz":"Nope"}]}`,
			`{"Statement":[{}]}`,
			nil,
			false,
		},
		{
			// Special shorthand for "all AWS accounts and anonymous"
			`{"Statement":[{"Principal":"*"}]}`,
			`{"Statement":[{"Principal":{"AWS":"*"}}]}`,
			nil,
			false,
		},
		{
			// Custom normalizer to eliminate "Resource"
			// Some AWS services insert the implied "Resource" value
			// describing the object the policy is attached to when
			// returning a policy document from the API. This is
			// redundant and can't be meaningfully set to any other
			// value in config, so we can remove it as part of
			// normalization to avoid spurious diffs.
			`{"Statement":[{"Effect":"Allow","Resource":"arn:fake:something"}]}`,
			`{"Statement":[{"Effect":"Allow"}]}`,
			func(stmt *IAMPolicyStatement) error {
				stmt.Resources = IAMPolicyStringSet{}
				return nil
			},
			false,
		},
		{
			// Invalid JSON suntax
			`{"Sta`,
			``,
			nil,
			true,
		},
		{
			// Inappropriate statement type
			`{"Statement":[true]}`,
			``,
			nil,
			true,
		},
		{
			// Inappropriate type for string set
			`{"Statement":[{"Action":[true]}]}`,
			``,
			nil,
			true,
		},
		{
			// Inappropriate type for string set redux
			`{"Statement":[{"Action":true}]}`,
			``,
			nil,
			true,
		},
		{
			// Principal string shorthand may only be used for wildcard
			`{"Statement":[{"Principal":"1234"}]}`,
			``,
			nil,
			true,
		},
	}

	for _, test := range tests {
		resultBytes, err := NormalizeIAMPolicyJSON([]byte(test.Input), test.Normalizer)

		if test.ExpectError {
			if err == nil {
				t.Errorf("%s normalized successfully; want error", test.Input)
				continue
			}

			result := string(resultBytes)
			if result != test.Input {
				t.Errorf("%s\nproduced %s\nshould match input", test.Input, result)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("%s returned error; want success\n%s", test.Input, err)
				continue
			}

			result := string(resultBytes)
			if result != test.Expected {
				t.Errorf("%s\nproduced %s\n    want %s", test.Input, result, test.Expected)
			}
		}
	}
}
