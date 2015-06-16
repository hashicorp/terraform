package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSSecurityGroupRuleMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			Attributes: map[string]string{
				// EBS
				"self": "true",
			},
			Expected: "sg-1234",
		},
		"v0_2": {
			StateVersion: 0,
			Attributes: map[string]string{
				// EBS
				"self": "false",
			},
			Expected: "sg-1235",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "sg-12333",
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsSecurityGroupRuleMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.ID != tc.Expected {
			t.Fatalf("bad sg rule id: %s\n\n expected: %s", is.ID, tc.Expected)
		}
	}
}
