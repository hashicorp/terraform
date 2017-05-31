package google

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestSqlUserMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
		ID           string
		ExpectedID   string
	}{
		"change id from $NAME to $INSTANCENAME.$NAME": {
			StateVersion: 0,
			Attributes: map[string]string{
				"name":     "tf-user",
				"instance": "tf-instance",
			},
			Expected: map[string]string{
				"name":     "tf-user",
				"instance": "tf-instance",
			},
			Meta:       &Config{},
			ID:         "tf-user",
			ExpectedID: "tf-instance/tf-user",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceSqlUserMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.ID != tc.ExpectedID {
			t.Fatalf("bad ID.\n\n expected: %s\n got: %s", tc.ExpectedID, is.ID)
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

func TestSqlUserMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta *Config

	// should handle nil
	is, err := resourceSqlUserMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceSqlUserMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
