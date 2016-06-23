package vsphere

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestVSphereVirtualMachineMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"skip_customization before 0.6.16": {
			StateVersion: 0,
			Attributes:   map[string]string{},
			Expected: map[string]string{
				"skip_customization": "false",
			},
		},
		"enable_disk_uuid before 0.6.16": {
			StateVersion: 0,
			Attributes:   map[string]string{},
			Expected: map[string]string{
				"enable_disk_uuid": "false",
			},
		},
		"disk controller_type": {
			StateVersion: 0,
			Attributes: map[string]string{
				"disk.1234.size":            "0",
				"disk.5678.size":            "0",
				"disk.9999.size":            "0",
				"disk.9999.controller_type": "ide",
			},
			Expected: map[string]string{
				"disk.1234.size":            "0",
				"disk.1234.controller_type": "scsi",
				"disk.5678.size":            "0",
				"disk.5678.controller_type": "scsi",
				"disk.9999.size":            "0",
				"disk.9999.controller_type": "ide",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		is, err := resourceVSphereVirtualMachineMigrateState(
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
	is, err := resourceVSphereVirtualMachineMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceVSphereVirtualMachineMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
