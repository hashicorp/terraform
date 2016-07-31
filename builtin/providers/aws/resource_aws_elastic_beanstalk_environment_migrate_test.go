package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSElasticBeanstalkEnvironmentMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0_1_web": {
			StateVersion: 0,
			Attributes: map[string]string{
				"tier": "",
			},
			Expected: map[string]string{
				"tier": "WebServer",
			},
		},
		"v0_1_web_explicit": {
			StateVersion: 0,
			Attributes: map[string]string{
				"tier": "WebServer",
			},
			Expected: map[string]string{
				"tier": "WebServer",
			},
		},
		"v0_1_worker": {
			StateVersion: 0,
			Attributes: map[string]string{
				"tier": "Worker",
			},
			Expected: map[string]string{
				"tier": "Worker",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "e-abcde12345",
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsElasticBeanstalkEnvironmentMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}
	}
}
