package google

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestGoogleProjectMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"deprecate policy_data and support creation/deletion": {
			StateVersion: 0,
			Attributes:   map[string]string{},
			Expected: map[string]string{
				"project_id":  "test-project",
				"skip_delete": "true",
			},
			Meta: &Config{},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "test-project",
			Attributes: tc.Attributes,
		}
		is, err := resourceGoogleProjectMigrateState(
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

func TestGoogleProjectMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta *Config

	// should handle nil
	is, err := resourceGoogleProjectMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceGoogleProjectMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
