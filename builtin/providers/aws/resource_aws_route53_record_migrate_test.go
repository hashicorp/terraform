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

func TestAWSRoute53RecordMigrateStateV1toV2(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 1,
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
