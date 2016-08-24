package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSSpotFleetRequestMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		ID           string
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			ID:           "some_id",
			Attributes: map[string]string{
				"associate_public_ip_address": "true",
			},
			Expected: "false",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsSpotFleetRequestMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["associate_public_ip_address"] != tc.Expected {
			t.Fatalf("bad Spot Fleet Request Migrate: %s\n\n expected: %s", is.Attributes["associate_public_ip_address"], tc.Expected)
		}
	}
}
