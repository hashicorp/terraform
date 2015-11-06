package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSKeyPairMigrateState(t *testing.T) {
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
				"fingerprint": "1d:cd:46:31:a9:4a:e0:06:8a:a1:22:cb:3b:bf:8e:42",
				"key_name":    "tf-testing-file",
				"public_key":  "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA4LBtwcFsQAYWw1cnOwRTZCJCzPSzq0dl3== ctshryock",
			},
			Expected: "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA4LBtwcFsQAYWw1cnOwRTZCJCzPSzq0dl3== ctshryock",
		},
		"v0_2": {
			StateVersion: 0,
			ID:           "tf-testing-file",
			Attributes: map[string]string{
				"fingerprint": "1d:cd:46:31:a9:4a:e0:06:8a:a1:22:cb:3b:bf:8e:42",
				"key_name":    "tf-testing-file",
				"public_key":  "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA4LBtwcFsQAYWw1cnOwRTZCJCzPSzq0dl3== ctshryock\n",
			},
			Expected: "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA4LBtwcFsQAYWw1cnOwRTZCJCzPSzq0dl3== ctshryock",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsKeyPairMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["public_key"] != tc.Expected {
			t.Fatalf("Bad public_key migration: %s\n\n expected: %s", is.Attributes["public_key"], tc.Expected)
		}
	}
}
