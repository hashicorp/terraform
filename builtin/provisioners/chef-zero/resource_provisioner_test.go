package chef

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"environment": "_default",
		"node_name":   "nodename1",
		"run_list":    []interface{}{"cookbook::recipe"},

		"local_nodes_dir": "nodes",
		"instance_id":     "myid",
		"user_name":       "bob",
		"user_key":        "USER-KEY",
		"dir_resources":   "test-fixtures",
	})

	warn, errs := Provisioner().Validate(c)
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

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

// Test that the JSON attributes with an unknown value don't
// validate.
func TestResourceProvider_Validate_computedValues(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"environment": "_default",
		"node_name":   "nodename1",
		"run_list":    []interface{}{"cookbook::recipe"},

		"user_name":            "bob",
		"user_key":             "USER-KEY",
		"instance_id":          "myid",
		"local_nodes_dir":      "nodes",
		"dna_attributes":       config.UnknownVariableValue,
		"automatic_attributes": config.UnknownVariableValue,
		"default_attributes":   config.UnknownVariableValue,
		"dir_resources":        "test-fixtures",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_runChefClient(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		ChefCmd  string
		ConfDir  string
		Commands map[string]bool
	}{
		"Sudo": {
			Config: map[string]interface{}{
				"node_name": "nodename1",
				"run_list":  []interface{}{"cookbook::recipe"},

				"environment": "_default",
				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
			},

			ChefCmd: linuxChefCmd,

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`sudo bash -c 'cd %s && %s -z -c %s -j %q -E "_default"'`,
					path.Join(linuxConfDir, "/cookbooks"),
					linuxChefCmd,
					path.Join(linuxConfDir, clienrb),
					path.Join(linuxConfDir, "dna", "myid.json")): true,
			},
		},

		"NoSudo": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"environment":  "_default",
				"user_name":    "bob",
				"instance_id":  "myid",

				"user_key": "USER-KEY",
			},

			ChefCmd: linuxChefCmd,

			ConfDir: linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`cd %s && %s -z -c %s -j %q -E "_default"`,
					path.Join(linuxConfDir, "/cookbooks"),
					linuxChefCmd,
					path.Join(linuxConfDir, clienrb),
					path.Join(linuxConfDir, "dna", "myid.json")): true,
			},
		},

		"Environment": {
			Config: map[string]interface{}{
				"environment":  "production",
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
			},

			ChefCmd: windowsChefCmd,

			ConfDir: windowsConfDir,

			Commands: map[string]bool{
				fmt.Sprintf(`cd %s && %s -z -c %s -j %q -E "production"`,
					path.Join(windowsConfDir, "/cookbooks"),
					windowsChefCmd,
					path.Join(windowsConfDir, clienrb),
					path.Join(windowsConfDir, "dna", "myid.json")): true,
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
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

func TestResourceProvider_configureVaults(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		GemCmd   string
		KnifeCmd string
		ConfDir  string
		Commands map[string]bool
	}{
		"Linux Vault string": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"instance_id":  "myid",

				"user_name":  "bob",
				"user_key":   "USER-KEY",
				"vault_json": `{"vault1": "item1"}`,
			},

			GemCmd:   linuxGemCmd,
			KnifeCmd: linuxKnifeCmd,
			ConfDir:  linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf("%s install chef-vault", linuxGemCmd): true,
				fmt.Sprintf("%s vault update vault1 item1 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
			},
		},

		"Linux Vault []string": {
			Config: map[string]interface{}{
				"fetch_chef_certificates": true,
				"node_name":               "nodename1",
				"prevent_sudo":            true,
				"run_list":                []interface{}{"cookbook::recipe"},

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key":   "USER-KEY",
				"vault_json": `{"vault1": ["item1", "item2"]}`,
			},

			GemCmd:   linuxGemCmd,
			KnifeCmd: linuxKnifeCmd,
			ConfDir:  linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf("%s install chef-vault", linuxGemCmd): true,
				fmt.Sprintf("%s vault update vault1 item1 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
				fmt.Sprintf("%s vault update vault1 item2 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
			},
		},

		"Linux Vault []string (recreate-client for vault)": {
			Config: map[string]interface{}{
				"fetch_chef_certificates": true,
				"node_name":               "nodename1",
				"prevent_sudo":            true,
				"run_list":                []interface{}{"cookbook::recipe"},

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key":        "USER-KEY",
				"vault_json":      `{"vault1": ["item1", "item2"]}`,
				"recreate_client": true,
			},

			GemCmd:   linuxGemCmd,
			KnifeCmd: linuxKnifeCmd,
			ConfDir:  linuxConfDir,

			Commands: map[string]bool{
				fmt.Sprintf("%s install chef-vault", linuxGemCmd): true,
				fmt.Sprintf("%s vault remove vault1 item1 -C \"nodename1\" -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
				fmt.Sprintf("%s vault remove vault1 item2 -C \"nodename1\" -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
				fmt.Sprintf("%s vault update vault1 item1 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
				fmt.Sprintf("%s vault update vault1 item2 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", linuxKnifeCmd, linuxConfDir, linuxConfDir): true,
			},
		},

		"Windows Vault string": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key":   "USER-KEY",
				"vault_json": `{"vault1": "item1"}`,
			},

			GemCmd:   windowsGemCmd,
			KnifeCmd: windowsKnifeCmd,
			ConfDir:  windowsConfDir,

			Commands: map[string]bool{
				fmt.Sprintf("%s install chef-vault", windowsGemCmd): true,
				fmt.Sprintf("%s vault update vault1 item1 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
			},
		},

		"Windows Vault []string": {
			Config: map[string]interface{}{
				"fetch_chef_certificates": true,
				"node_name":               "nodename1",
				"prevent_sudo":            true,
				"run_list":                []interface{}{"cookbook::recipe"},

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key":   "USER-KEY",
				"vault_json": `{"vault1": ["item1", "item2"]}`,
			},

			GemCmd:   windowsGemCmd,
			KnifeCmd: windowsKnifeCmd,
			ConfDir:  windowsConfDir,

			Commands: map[string]bool{
				fmt.Sprintf("%s install chef-vault", windowsGemCmd): true,
				fmt.Sprintf("%s vault update vault1 item1 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
				fmt.Sprintf("%s vault update vault1 item2 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
			},
		},

		"Windows Vault [] string (recreate-client for vault)": {
			Config: map[string]interface{}{
				"fetch_chef_certificates": true,
				"node_name":               "nodename1",
				"prevent_sudo":            true,
				"run_list":                []interface{}{"cookbook::recipe"},

				"user_name":       "bob",
				"instance_id":     "myid",
				"user_key":        "USER-KEY",
				"vault_json":      `{"vault1": ["item1", "item2"]}`,
				"recreate_client": true,
			},

			GemCmd:   windowsGemCmd,
			KnifeCmd: windowsKnifeCmd,
			ConfDir:  windowsConfDir,

			Commands: map[string]bool{
				fmt.Sprintf("%s install chef-vault", windowsGemCmd): true,
				fmt.Sprintf("%s vault remove vault1 item1 -C \"nodename1\" -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
				fmt.Sprintf("%s vault remove vault1 item2 -C \"nodename1\" -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
				fmt.Sprintf("%s vault update vault1 item1 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
				fmt.Sprintf("%s vault update vault1 item2 -C nodename1 -M client -c %s/client.rb "+
					"-u bob --key %s/bob.pem", windowsKnifeCmd, windowsConfDir, windowsConfDir): true,
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)

		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.configureVaults = p.configureVaultsFunc(tc.GemCmd, tc.KnifeCmd, tc.ConfDir)
		p.useSudo = !p.PreventSudo

		err = p.configureVaults(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
