package chef

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = new(ResourceProvisioner)
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"attributes":             []interface{}{"key1 { subkey1 = value1 }"},
		"environment":            "_default",
		"node_name":              "nodename1",
		"run_list":               []interface{}{"cookbook::recipe"},
		"server_url":             "https://chef.local",
		"validation_client_name": "validator",
		"validation_key":         "contentsofsomevalidator.pem",
	})
	r := new(ResourceProvisioner)
	warn, errs := r.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_bad(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"invalid": "nope",
	})
	p := new(ResourceProvisioner)
	warn, errs := p.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}

func TestResourceProvider_runChefClient(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		ChefCmd  string
		ConfDir  string
		Commands map[string]bool
	}{
		"Sudo": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			ChefCmd: linuxChefCmd,

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`sudo %s -j %q -E "_default"`,
					linuxChefCmd,
					path.Join(linuxConfDir, "first-boot.json")): true,
			},
		},

		"NoSudo": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			ChefCmd: linuxChefCmd,

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`%s -j %q -E "_default"`,
					linuxChefCmd,
					path.Join(linuxConfDir, "first-boot.json")): true,
			},
		},

		"Environment": {
			Config: testConfig(t, map[string]interface{}{
				"environment":            "production",
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			ChefCmd: windowsChefCmd,

			ConfDir: windowsConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`%s -j %q -E "production"`,
					windowsChefCmd,
					path.Join(windowsConfDir, "first-boot.json")): true,
			},
		},
	}

	r := new(ResourceProvisioner)
	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands

		p, err := r.decodeConfig(tc.Config)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.runChefClient = p.runChefClientFunc(tc.ChefCmd, tc.ConfDir)
		p.useSudo = !p.PreventSudo

		err = p.runChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}
