package google

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestComputeInstanceMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0.4.2 and earlier": {
			StateVersion: 0,
			Attributes: map[string]string{
				"metadata.#":           "2",
				"metadata.0.foo":       "bar",
				"metadata.1.baz":       "qux",
				"metadata.2.with.dots": "should.work",
			},
			Expected: map[string]string{
				"metadata.foo":       "bar",
				"metadata.baz":       "qux",
				"metadata.with.dots": "should.work",
			},
		},
		"change scope from list to set": {
			StateVersion: 1,
			Attributes: map[string]string{
				"service_account.#":          "1",
				"service_account.0.email":    "xxxxxx-compute@developer.gserviceaccount.com",
				"service_account.0.scopes.#": "4",
				"service_account.0.scopes.0": "https://www.googleapis.com/auth/compute",
				"service_account.0.scopes.1": "https://www.googleapis.com/auth/datastore",
				"service_account.0.scopes.2": "https://www.googleapis.com/auth/devstorage.full_control",
				"service_account.0.scopes.3": "https://www.googleapis.com/auth/logging.write",
			},
			Expected: map[string]string{
				"service_account.#":                   "1",
				"service_account.0.email":             "xxxxxx-compute@developer.gserviceaccount.com",
				"service_account.0.scopes.#":          "4",
				"service_account.0.scopes.1693978638": "https://www.googleapis.com/auth/devstorage.full_control",
				"service_account.0.scopes.172152165":  "https://www.googleapis.com/auth/logging.write",
				"service_account.0.scopes.299962681":  "https://www.googleapis.com/auth/compute",
				"service_account.0.scopes.3435931483": "https://www.googleapis.com/auth/datastore",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		is, err := resourceComputeInstanceMigrateState(
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

func TestComputeInstanceMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta interface{}

	// should handle nil
	is, err := resourceComputeInstanceMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceComputeInstanceMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
