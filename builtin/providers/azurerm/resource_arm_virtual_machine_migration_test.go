package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAzureRMVirtualMachineMigrateStateV0ToV1(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0_1_set_default": {
			StateVersion: 0,
			Attributes: map[string]string{
				"os_profile_windows_config.#":                                       "1",
				"os_profile_windows_config.2256145325.additional_unattend_config.#": "0",
				"os_profile_windows_config.2256145325.enable_automatic_upgrades":    "true",
				"os_profile_windows_config.2256145325.provision_vm_agent":           "true",
				"os_profile_windows_config.2256145325.winrm.#":                      "0",
			},
			Expected: map[string]string{
				"os_profile_windows_config.#":                                       "1",
				"os_profile_windows_config.2256145325.additional_unattend_config.#": "0",
				"os_profile_windows_config.2256145325.enable_automatic_upgrades":    "true",
				"os_profile_windows_config.2256145325.provision_vm_agent":           "true",
				"os_profile_windows_config.2256145325.winrm.#":                      "0",
			},
		},
		"v0_1_set_other": {
			StateVersion: 0,
			Attributes: map[string]string{
				"os_profile_windows_config.#":                                      "1",
				"os_profile_windows_config.429474957.additional_unattend_config.#": "0",
				"os_profile_windows_config.429474957.enable_automatic_upgrades":    "false",
				"os_profile_windows_config.429474957.provision_vm_agent":           "false",
				"os_profile_windows_config.429474957.winrm.#":                      "0",
			},
			Expected: map[string]string{
				"os_profile_windows_config.#":                                      "1",
				"os_profile_windows_config.429474957.additional_unattend_config.#": "0",
				"os_profile_windows_config.429474957.enable_automatic_upgrades":    "false",
				"os_profile_windows_config.429474957.provision_vm_agent":           "false",
				"os_profile_windows_config.429474957.winrm.#":                      "0",
			},
		},
		"v0_1_empty": {
			StateVersion: 0,
			Attributes:   map[string]string{},
			Expected:     map[string]string{},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "azurerm_virtual_machine",
			Attributes: tc.Attributes,
		}
		is, err := resourceArmVirtualMachine().MigrateState(tc.StateVersion, is, tc.Meta)

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
