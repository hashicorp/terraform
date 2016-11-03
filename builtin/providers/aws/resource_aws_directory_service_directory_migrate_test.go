package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAwsDirectoryServicesDirectoryMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		ID           string
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			ID:           "d-abc123",
			Attributes: map[string]string{
				"password": "hunter2",
			},
			Expected: "f52fbd32b2b3b86ff88ef6c490628285f482af15ddcb29541f94bcf526a3f6c7",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsDirectoryServiceDirectoryMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["password"] != tc.Expected {
			t.Fatalf("Bad password hash migration: %s\n\n expected %s", is.Attributes["password"], tc.Expected)
		}
	}
}
