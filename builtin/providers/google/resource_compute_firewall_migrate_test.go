package google

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestComputeFirewallMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"change scope from list to set": {
			StateVersion: 0,
			Attributes: map[string]string{
				"allow.#":                  "1",
				"allow.0.protocol":         "udp",
				"allow.0.ports.#":          "4",
				"allow.0.ports.1693978638": "8080",
				"allow.0.ports.172152165":  "8081",
				"allow.0.ports.299962681":  "7072",
				"allow.0.ports.3435931483": "4044",
			},
			Expected: map[string]string{
				"allow.#":          "1",
				"allow.0.protocol": "udp",
				"allow.0.ports.#":  "4",
				"allow.0.ports.0":  "8080",
				"allow.0.ports.1":  "8081",
				"allow.0.ports.2":  "7072",
				"allow.0.ports.3":  "4044",
			},
		},
	}
	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		is, err := resourceComputeFirewallMigrateState(
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

func TestComputeFirewallMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta interface{}

	// should handle nil
	is, err := resourceComputeFirewallMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceComputeFirewallMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
