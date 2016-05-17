package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSRoute53RecordMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		ID           string
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_0": {
			StateVersion: 0,
			ID:           "some_id",
			Attributes: map[string]string{
				"name": "www",
			},
			Expected: "www",
		},
		"v0_1": {
			StateVersion: 0,
			ID:           "some_id",
			Attributes: map[string]string{
				"name": "www.notdomain.com.",
			},
			Expected: "www.notdomain.com",
		},
		"v0_2": {
			StateVersion: 0,
			ID:           "some_id",
			Attributes: map[string]string{
				"name": "www.notdomain.com",
			},
			Expected: "www.notdomain.com",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsRoute53RecordMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["name"] != tc.Expected {
			t.Fatalf("bad Route 53 Migrate: %s\n\n expected: %s", is.Attributes["name"], tc.Expected)
		}
	}
}
