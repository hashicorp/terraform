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
				Config:           testConfigStrProvider,
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

func TestTest_importStateFail(t *testing.T) {
	mp := testProvider()
	mp.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID:        "bar",
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
				Config:           testConfigStrProvider,
				ResourceName:     "test_instance.foo",
				ImportState:      true,
				ImportStateId:    "foo",
				ImportStateCheck: checkFn,
			},
		},
	})

	if !mt.failed() {
		t.Fatal("should fail")
	}
	if !checked {
		t.Fatal("didn't call check")
	}
}

func TestTest_importStateDetectId(t *testing.T) {
	mp := testProvider()
	mp.DiffReturn = nil
	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
			}, nil
		}

		return nil, nil
	}

	mp.RefreshFn = func(
		i *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		return s, nil
	}

	mp.ImportStateFn = func(
		info *terraform.InstanceInfo, id string) ([]*terraform.InstanceState, error) {
		if id != "foo" {
			return nil, fmt.Errorf("bad import ID: %s", id)
		}

		return []*terraform.InstanceState{
			&terraform.InstanceState{
				ID:        "bar",
				Ephemeral: terraform.EphemeralState{Type: "test_instance"},
			},
		}, nil
	}

	checked := false
	checkFn := func(s []*terraform.InstanceState) error {
		checked = true

		if s[0].ID != "bar" {
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
				Config: testConfigStr,
			},
			TestStep{
				Config:           testConfigStr,
				ResourceName:     "test_instance.foo",
				ImportState:      true,
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

func TestTest_importStateIdPrefix(t *testing.T) {
	mp := testProvider()
	mp.DiffReturn = nil
	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
			}, nil
		}

		return nil, nil
	}

	mp.RefreshFn = func(
		i *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		return s, nil
	}

	mp.ImportStateFn = func(
		info *terraform.InstanceInfo, id string) ([]*terraform.InstanceState, error) {
		if id != "bazfoo" {
			return nil, fmt.Errorf("bad import ID: %s", id)
		}

		return []*terraform.InstanceState{
			{
				ID:        "bar",
				Ephemeral: terraform.EphemeralState{Type: "test_instance"},
			},
		}, nil
	}

	checked := false
	checkFn := func(s []*terraform.InstanceState) error {
		checked = true

		if s[0].ID != "bar" {
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
			{
				Config: testConfigStr,
			},
			{
				Config:              testConfigStr,
				ResourceName:        "test_instance.foo",
				ImportState:         true,
				ImportStateCheck:    checkFn,
				ImportStateIdPrefix: "baz",
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

func TestTest_importStateVerify(t *testing.T) {
	mp := testProvider()
	mp.DiffReturn = nil
	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
				Attributes: map[string]string{
					"foo": "bar",
				},
			}, nil
		}

		return nil, nil
	}

	mp.RefreshFn = func(
		i *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		if len(s.Attributes) == 0 {
			s.Attributes = map[string]string{
				"id":  s.ID,
				"foo": "bar",
			}
		}

		return s, nil
	}

	mp.ImportStateFn = func(
		info *terraform.InstanceInfo, id string) ([]*terraform.InstanceState, error) {
		if id != "foo" {
			return nil, fmt.Errorf("bad import ID: %s", id)
		}

		return []*terraform.InstanceState{
			&terraform.InstanceState{
				ID:        "foo",
				Ephemeral: terraform.EphemeralState{Type: "test_instance"},
			},
		}, nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},

		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
			},
			TestStep{
				Config:            testConfigStr,
				ResourceName:      "test_instance.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	if mt.failed() {
		t.Fatalf("test failed: %s", mt.failMessage())
	}
}

func TestTest_importStateVerifyFail(t *testing.T) {
	mp := testProvider()
	mp.DiffReturn = nil
	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
				Attributes: map[string]string{
					"foo": "bar",
				},
			}, nil
		}

		return nil, nil
	}

	mp.RefreshFn = func(
		i *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		return s, nil
	}

	mp.ImportStateFn = func(
		info *terraform.InstanceInfo, id string) ([]*terraform.InstanceState, error) {
		if id != "foo" {
			return nil, fmt.Errorf("bad import ID: %s", id)
		}

		return []*terraform.InstanceState{
			&terraform.InstanceState{
				ID:        "foo",
				Ephemeral: terraform.EphemeralState{Type: "test_instance"},
			},
		}, nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},

		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
			},
			TestStep{
				Config:            testConfigStr,
				ResourceName:      "test_instance.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	if !mt.failed() {
		t.Fatalf("test should fail")
	}
}

func TestTest_importStateIdFunc(t *testing.T) {
	mp := testProvider()
	mp.ImportStateFn = func(
		info *terraform.InstanceInfo, id string) ([]*terraform.InstanceState, error) {
		if id != "foo:bar" {
			return nil, fmt.Errorf("bad import ID: %s", id)
		}

		return []*terraform.InstanceState{
			{
				ID:        "foo",
				Ephemeral: terraform.EphemeralState{Type: "test_instance"},
			},
		}, nil
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
				Config:            testConfigStrProvider,
				ResourceName:      "test_instance.foo",
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return "foo:bar", nil },
				ImportStateCheck:  checkFn,
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
