package chef

import (
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
		"validation_key_path":    "validator.pem",
		"secret_key_path":    	  "encrypted_data_bag_secret",
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
				"secret_key_path":    "test-fixtures/encrypted_data_bag_secret",
			}),

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				`sudo chef-client -j "/etc/chef/first-boot.json" -E "_default"`: true,
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
				"secret_key_path":    "test-fixtures/encrypted_data_bag_secret",
			}),

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				`chef-client -j "/etc/chef/first-boot.json" -E "_default"`: true,
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
				"secret_key_path":    "test-fixtures/encrypted_data_bag_secret",
			}),

			ConfDir: windowsConfDir,

			Commands: map[string]bool{
				`chef-client -j "C:/chef/first-boot.json" -E "production"`: true,
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

		p.runChefClient = p.runChefClientFunc(tc.ConfDir)
		p.useSudo = !p.PreventSudo

		err = p.runChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}
