package resource

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestTest_importState(t *testing.T) {
	mp := testProvider()
	mp.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID:        "foo",
			Ephemeral: terraform.EphemeralState{Type: "test_instance"},
		},
	}
	mp.RefreshFn = func(
		i *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		return s, nil
	}

	checked := false
	checkFn := func(s []*terraform.InstanceState) error {
		checked = true

		if s[0].ID != "foo" {
			return fmt.Errorf("bad: %#v", s)
		}

		return nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},

		Steps: []TestStep{
			TestStep{
				ResourceName:     "test_instance.foo",
				ImportState:      true,
				ImportStateId:    "foo",
				ImportStateCheck: checkFn,
			},
		},
	})

	if mt.failed() {
		t.Fatalf("test failed: %s", mt.failMessage())
	}
	if !checked {
		t.Fatal("didn't call check")
	}
}
