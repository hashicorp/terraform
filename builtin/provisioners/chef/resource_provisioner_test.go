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

func TestResourceProvider_fetchChefCertificates(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		KnifeCmd string
		ConfDir  string
		Commands map[string]bool
	}{
		"Sudo": {
			Config: testConfig(t, map[string]interface{}{
				"fetch_chef_certificates": true,
				"node_name":               "nodename1",
				"run_list":                []interface{}{"cookbook::recipe"},
				"server_url":              "https://chef.local",
				"validation_client_name":  "validator",
				"validation_key_path":     "test-fixtures/validator.pem",
			}),

			KnifeCmd: linuxKnifeCmd,

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`sudo %s ssl fetch -s https://chef.local`,
					linuxKnifeCmd): true,
			},
		},

		"NoSudo": {
			Config: testConfig(t, map[string]interface{}{
				"fetch_chef_certificates": true,
				"node_name":               "nodename1",
				"prevent_sudo":            true,
				"run_list":                []interface{}{"cookbook::recipe"},
				"server_url":              "https://chef.local",
				"validation_client_name":  "validator",
				"validation_key_path":     "test-fixtures/validator.pem",
			}),

			KnifeCmd: windowsKnifeCmd,

			ConfDir: windowsConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`%s ssl fetch -s https://chef.local`,
					windowsKnifeCmd): true,
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

		p.fetchChefCertificates = p.fetchChefCertificatesFunc(tc.KnifeCmd, tc.ConfDir)
		p.useSudo = !p.PreventSudo

		err = p.fetchChefCertificates(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestResourceProvider_chefBootstrap(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		KnifeCmd string
		TmpDir   string
		Commands map[string]bool
	}{
		"SimpleBootstrap": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			KnifeCmd: linuxKnifeCmd,

			TmpDir: linuxTmpDir,

			Commands: map[string]bool{
				fmt.Sprintf(`sudo %s bootstrap 127.0.0.1 -y -x $(whoami) -i /root/.ssh/id_rsa -u validator -k /tmp/validation.pem -N nodename1 --server-url https://chef.local -r cookbook::recipe -E _default --sudo`,
					linuxKnifeCmd): true,
				fmt.Sprintf(`sudo cp /home/$(whoami)/.ssh/authorized_keys.pre_bootstrap /home/$(whoami)/.ssh/authorized_keys`):      true,
				fmt.Sprintf(`sudo rm -f /home/$(whoami)/.ssh/authorized_keys.pre_bootstrap /root/.ssh/id_rsa* /tmp/validation.pem`): true,
			},
		},

		"ComplexBootstrap": {
			Config: testConfig(t, map[string]interface{}{
				"attributes_json":        "{\"test\":\"test_val\"}",
				"environment":            "prod",
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe", "cookbook::recipe2"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
				"vaults_json":            "{\"vault_test\":[\"test_vault_item\"]}",
			}),

			KnifeCmd: linuxKnifeCmd,

			TmpDir: linuxTmpDir,

			Commands: map[string]bool{
				fmt.Sprintf(`sudo %s bootstrap 127.0.0.1 -y -x $(whoami) -i /root/.ssh/id_rsa -u validator -k /tmp/validation.pem -N nodename1 --server-url https://chef.local -r cookbook::recipe,cookbook::recipe2 -E prod -j '{"test":"test_val"}' --bootstrap-vault-json '{"vault_test":["test_vault_item"]}' --sudo`,
					linuxKnifeCmd): true,
				fmt.Sprintf(`sudo cp /home/$(whoami)/.ssh/authorized_keys.pre_bootstrap /home/$(whoami)/.ssh/authorized_keys`):      true,
				fmt.Sprintf(`sudo rm -f /home/$(whoami)/.ssh/authorized_keys.pre_bootstrap /root/.ssh/id_rsa* /tmp/validation.pem`): true,
			},
		},

		"EmptyRunlistBootstrap": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			KnifeCmd: linuxKnifeCmd,

			TmpDir: linuxTmpDir,

			Commands: map[string]bool{
				fmt.Sprintf(`sudo %s bootstrap 127.0.0.1 -y -x $(whoami) -i /root/.ssh/id_rsa -u validator -k /tmp/validation.pem -N nodename1 --server-url https://chef.local -E _default --sudo`,
					linuxKnifeCmd): true,
				fmt.Sprintf(`sudo cp /home/$(whoami)/.ssh/authorized_keys.pre_bootstrap /home/$(whoami)/.ssh/authorized_keys`):      true,
				fmt.Sprintf(`sudo rm -f /home/$(whoami)/.ssh/authorized_keys.pre_bootstrap /root/.ssh/id_rsa* /tmp/validation.pem`): true,
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

		p.bootstrapCleanup = p.linuxBootstrapCleanup
		p.chefBootstrap = p.chefBootstrapFunc(tc.KnifeCmd, tc.TmpDir)
		p.useSudo = !p.PreventSudo

		err = p.chefBootstrap(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}
