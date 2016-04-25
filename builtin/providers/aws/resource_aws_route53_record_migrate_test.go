package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSRoute53RecordMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			Attributes: map[string]string{
				"weight":   "0",
				"failover": "PRIMARY",
			},
			Expected: map[string]string{
				"weighted_routing_policy.#":        "1",
				"weighted_routing_policy.0.weight": "0",
				"failover_routing_policy.#":        "1",
				"failover_routing_policy.0.type":   "PRIMARY",
			},
		},
		"v0_2": {
			StateVersion: 0,
			Attributes: map[string]string{
				"weight": "-1",
			},
			Expected: map[string]string{},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "route53_record",
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsRoute53Record().MigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		for k, v := range tc.Expected {
			if is.Attributes[k] != v {
				t.Fatalf(
					"bad: %s\n\n expected: %#v -> %#v\n got: %#v -> %#v\n in: %#v",
					tn, k, v, k, is.Attributes[k], is.Attributes)
			}
		}
	}
}
