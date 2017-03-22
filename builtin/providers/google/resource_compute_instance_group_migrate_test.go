package google

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestComputeInstanceGroupMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"change instances from list to set": {
			StateVersion: 0,
			Attributes: map[string]string{
				"instances.#": "1",
				"instances.0": "https://www.googleapis.com/compute/v1/projects/project_name/zones/zone_name/instances/instancegroup-test-1",
				"instances.1": "https://www.googleapis.com/compute/v1/projects/project_name/zones/zone_name/instances/instancegroup-test-0",
			},
			Expected: map[string]string{
				"instances.#":          "1",
				"instances.764135222":  "https://www.googleapis.com/compute/v1/projects/project_name/zones/zone_name/instances/instancegroup-test-1",
				"instances.1519187872": "https://www.googleapis.com/compute/v1/projects/project_name/zones/zone_name/instances/instancegroup-test-0",
			},
			Meta: &Config{},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		is, err := resourceComputeInstanceGroupMigrateState(
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

func TestComputeInstanceGroupMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta *Config

	// should handle nil
	is, err := resourceComputeInstanceGroupMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceComputeInstanceGroupMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
