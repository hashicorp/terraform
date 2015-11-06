package chef

import (
	"path"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_linuxInstallChefClient(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		Commands map[string]bool
	}{
		"Sudo": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
			}),

			Commands: map[string]bool{
				"sudo curl -LO https://www.chef.io/chef/install.sh": true,
				"sudo bash ./install.sh -v \"\"":                    true,
				"sudo rm -f install.sh":                             true,
			},
		},

		"NoSudo": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
				"secret_key_path":        "encrypted_data_bag_secret",
			}),

			Commands: map[string]bool{
				"curl -LO https://www.chef.io/chef/install.sh": true,
				"bash ./install.sh -v \"\"":                    true,
				"rm -f install.sh":                             true,
			},
		},

		"HTTPProxy": {
			Config: testConfig(t, map[string]interface{}{
				"http_proxy":             "http://proxy.local",
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
			}),

			Commands: map[string]bool{
				"http_proxy='http://proxy.local' curl -LO https://www.chef.io/chef/install.sh": true,
				"http_proxy='http://proxy.local' bash ./install.sh -v \"\"":                    true,
				"http_proxy='http://proxy.local' rm -f install.sh":                             true,
			},
		},

		"HTTPSProxy": {
			Config: testConfig(t, map[string]interface{}{
				"https_proxy":            "https://proxy.local",
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
			}),

			Commands: map[string]bool{
				"https_proxy='https://proxy.local' curl -LO https://www.chef.io/chef/install.sh": true,
				"https_proxy='https://proxy.local' bash ./install.sh -v \"\"":                    true,
				"https_proxy='https://proxy.local' rm -f install.sh":                             true,
			},
		},

		"NoProxy": {
			Config: testConfig(t, map[string]interface{}{
				"http_proxy":             "http://proxy.local",
				"no_proxy":               []interface{}{"http://local.local", "http://local.org"},
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
			}),

			Commands: map[string]bool{
				"http_proxy='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"curl -LO https://www.chef.io/chef/install.sh": true,
				"http_proxy='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"bash ./install.sh -v \"\"": true,
				"http_proxy='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"rm -f install.sh": true,
			},
		},

		"Version": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
				"version":                "11.18.6",
			}),

			Commands: map[string]bool{
				"curl -LO https://www.chef.io/chef/install.sh": true,
				"bash ./install.sh -v \"11.18.6\"":             true,
				"rm -f install.sh":                             true,
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

		p.useSudo = !p.PreventSudo

		err = p.linuxInstallChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestResourceProvider_linuxCreateConfigFiles(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Sudo": {
			Config: testConfig(t, map[string]interface{}{
				"ohai_hints":             []interface{}{"test-fixtures/ohaihint.json"},
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"secret_key_path":        "test-fixtures/encrypted_data_bag_secret",
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"sudo mkdir -p " + linuxConfDir:                                    true,
				"sudo chmod 777 " + linuxConfDir:                                   true,
				"sudo mkdir -p " + path.Join(linuxConfDir, "ohai/hints"):           true,
				"sudo chmod 777 " + path.Join(linuxConfDir, "ohai/hints"):          true,
				"sudo chmod 755 " + path.Join(linuxConfDir, "ohai/hints"):          true,
				"sudo chown -R root.root " + path.Join(linuxConfDir, "ohai/hints"): true,
				"sudo chmod 755 " + linuxConfDir:                                   true,
				"sudo chown -R root.root " + linuxConfDir:                          true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY-FILE",
				linuxConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/ohai/hints/ohaihint.json":  "OHAI-HINT-FILE",
				linuxConfDir + "/validation.pem":            "VALIDATOR-PEM-FILE",
			},
		},

		"NoSudo": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"secret_key_path":        "test-fixtures/encrypted_data_bag_secret",
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY-FILE",
				linuxConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/validation.pem":            "VALIDATOR-PEM-FILE",
			},
		},

		"Proxy": {
			Config: testConfig(t, map[string]interface{}{
				"http_proxy":             "http://proxy.local",
				"https_proxy":            "https://proxy.local",
				"no_proxy":               []interface{}{"http://local.local", "https://local.local"},
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"secret_key_path":        "test-fixtures/encrypted_data_bag_secret",
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 proxyLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY-FILE",
				linuxConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/validation.pem":            "VALIDATOR-PEM-FILE",
			},
		},

		"Attributes": {
			Config: testConfig(t, map[string]interface{}{
				"attributes": []map[string]interface{}{
					map[string]interface{}{
						"key1": []map[string]interface{}{
							map[string]interface{}{
								"subkey1": []map[string]interface{}{
									map[string]interface{}{
										"subkey2a": []interface{}{
											"val1", "val2", "val3",
										},
										"subkey2b": []map[string]interface{}{
											map[string]interface{}{
												"subkey3": "value3",
											},
										},
									},
								},
							},
						},
						"key2": "value2",
					},
				},
				"node_name":              "nodename1",
				"prevent_sudo":           true,
				"run_list":               []interface{}{"cookbook::recipe"},
				"secret_key_path":        "test-fixtures/encrypted_data_bag_secret",
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY-FILE",
				linuxConfDir + "/validation.pem":            "VALIDATOR-PEM-FILE",
				linuxConfDir + "/first-boot.json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3"}}},"key2":"value2","run_list":["cookbook::recipe"]}`,
			},
		},
	}

	r := new(ResourceProvisioner)
	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads

		p, err := r.decodeConfig(tc.Config)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.useSudo = !p.PreventSudo

		err = p.linuxCreateConfigFiles(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

const defaultLinuxClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local"
validation_client_name  "validator"
node_name               "nodename1"`

const proxyLinuxClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local"
validation_client_name  "validator"
node_name               "nodename1"




http_proxy          "http://proxy.local"
ENV['http_proxy'] = "http://proxy.local"
ENV['HTTP_PROXY'] = "http://proxy.local"



https_proxy          "https://proxy.local"
ENV['https_proxy'] = "https://proxy.local"
ENV['HTTPS_PROXY'] = "https://proxy.local"


no_proxy "http://local.local,https://local.local"`
