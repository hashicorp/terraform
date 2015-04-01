package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSInstanceMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0.3.6 and earlier": {
			StateVersion: 0,
			Attributes: map[string]string{
				// EBS
				"block_device.#":                                "2",
				"block_device.3851383343.delete_on_termination": "true",
				"block_device.3851383343.device_name":           "/dev/sdx",
				"block_device.3851383343.encrypted":             "false",
				"block_device.3851383343.snapshot_id":           "",
				"block_device.3851383343.virtual_name":          "",
				"block_device.3851383343.volume_size":           "5",
				"block_device.3851383343.volume_type":           "standard",
				// Ephemeral
				"block_device.3101711606.delete_on_termination": "false",
				"block_device.3101711606.device_name":           "/dev/sdy",
				"block_device.3101711606.encrypted":             "false",
				"block_device.3101711606.snapshot_id":           "",
				"block_device.3101711606.virtual_name":          "ephemeral0",
				"block_device.3101711606.volume_size":           "",
				"block_device.3101711606.volume_type":           "",
				// Root
				"block_device.56575650.delete_on_termination": "true",
				"block_device.56575650.device_name":           "/dev/sda1",
				"block_device.56575650.encrypted":             "false",
				"block_device.56575650.snapshot_id":           "",
				"block_device.56575650.volume_size":           "10",
				"block_device.56575650.volume_type":           "standard",
			},
			Expected: map[string]string{
				"ebs_block_device.#":                                 "1",
				"ebs_block_device.3851383343.delete_on_termination":  "true",
				"ebs_block_device.3851383343.device_name":            "/dev/sdx",
				"ebs_block_device.3851383343.encrypted":              "false",
				"ebs_block_device.3851383343.snapshot_id":            "",
				"ebs_block_device.3851383343.volume_size":            "5",
				"ebs_block_device.3851383343.volume_type":            "standard",
				"ephemeral_block_device.#":                           "1",
				"ephemeral_block_device.2458403513.device_name":      "/dev/sdy",
				"ephemeral_block_device.2458403513.virtual_name":     "ephemeral0",
				"root_block_device.#":                                "1",
				"root_block_device.3018388612.delete_on_termination": "true",
				"root_block_device.3018388612.device_name":           "/dev/sda1",
				"root_block_device.3018388612.snapshot_id":           "",
				"root_block_device.3018388612.volume_size":           "10",
				"root_block_device.3018388612.volume_type":           "standard",
			},
		},
		"v0.3.7": {
			StateVersion: 0,
			Attributes: map[string]string{
				// EBS
				"block_device.#":                                "2",
				"block_device.3851383343.delete_on_termination": "true",
				"block_device.3851383343.device_name":           "/dev/sdx",
				"block_device.3851383343.encrypted":             "false",
				"block_device.3851383343.snapshot_id":           "",
				"block_device.3851383343.virtual_name":          "",
				"block_device.3851383343.volume_size":           "5",
				"block_device.3851383343.volume_type":           "standard",
				"block_device.3851383343.iops":                  "",
				// Ephemeral
				"block_device.3101711606.delete_on_termination": "false",
				"block_device.3101711606.device_name":           "/dev/sdy",
				"block_device.3101711606.encrypted":             "false",
				"block_device.3101711606.snapshot_id":           "",
				"block_device.3101711606.virtual_name":          "ephemeral0",
				"block_device.3101711606.volume_size":           "",
				"block_device.3101711606.volume_type":           "",
				"block_device.3101711606.iops":                  "",
				// Root
				"root_block_device.#":                                "1",
				"root_block_device.3018388612.delete_on_termination": "true",
				"root_block_device.3018388612.device_name":           "/dev/sda1",
				"root_block_device.3018388612.snapshot_id":           "",
				"root_block_device.3018388612.volume_size":           "10",
				"root_block_device.3018388612.volume_type":           "io1",
				"root_block_device.3018388612.iops":                  "1000",
			},
			Expected: map[string]string{
				"ebs_block_device.#":                                 "1",
				"ebs_block_device.3851383343.delete_on_termination":  "true",
				"ebs_block_device.3851383343.device_name":            "/dev/sdx",
				"ebs_block_device.3851383343.encrypted":              "false",
				"ebs_block_device.3851383343.snapshot_id":            "",
				"ebs_block_device.3851383343.volume_size":            "5",
				"ebs_block_device.3851383343.volume_type":            "standard",
				"ephemeral_block_device.#":                           "1",
				"ephemeral_block_device.2458403513.device_name":      "/dev/sdy",
				"ephemeral_block_device.2458403513.virtual_name":     "ephemeral0",
				"root_block_device.#":                                "1",
				"root_block_device.3018388612.delete_on_termination": "true",
				"root_block_device.3018388612.device_name":           "/dev/sda1",
				"root_block_device.3018388612.snapshot_id":           "",
				"root_block_device.3018388612.volume_size":           "10",
				"root_block_device.3018388612.volume_type":           "io1",
				"root_block_device.3018388612.iops":                  "1000",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsInstanceMigrateState(
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

func TestAWSInstanceMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta interface{}

	// should handle nil
	is, err := resourceAwsInstanceMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceAwsInstanceMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
