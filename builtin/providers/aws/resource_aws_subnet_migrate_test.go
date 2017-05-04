package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSSubnetMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		ID           string
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1_without_value": {
			StateVersion: 0,
			ID:           "some_id",
			Attributes:   map[string]string{},
			Expected:     "false",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsSubnetMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["assign_ipv6_address_on_creation"] != tc.Expected {
			t.Fatalf("bad Subnet Migrate: %s\n\n expected: %s", is.Attributes["assign_ipv6_address_on_creation"], tc.Expected)
		}
	}
}
