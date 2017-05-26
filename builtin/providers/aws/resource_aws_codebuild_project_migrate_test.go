package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSCodebuildMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		ID           string
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			ID:           "tf-testing-file",
			Attributes: map[string]string{
				"description": "some description",
				"timeout":     "5",
			},
			Expected: "5",
		},
		"v0_2": {
			StateVersion: 0,
			ID:           "tf-testing-file",
			Attributes: map[string]string{
				"description":   "some description",
				"build_timeout": "5",
			},
			Expected: "5",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsCodebuildMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["build_timeout"] != tc.Expected {
			t.Fatalf("Bad build_timeout migration: %s\n\n expected: %s", is.Attributes["build_timeout"], tc.Expected)
		}
	}
}
