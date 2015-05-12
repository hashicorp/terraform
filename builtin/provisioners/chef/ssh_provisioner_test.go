package chef

import (
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_sshInstallChefClient(t *testing.T) {
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
				"sudo bash ./install.sh -v ":                        true,
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
			}),

			Commands: map[string]bool{
				"curl -LO https://www.chef.io/chef/install.sh": true,
				"bash ./install.sh -v ":                        true,
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
				"proxy_http='http://proxy.local' curl -LO https://www.chef.io/chef/install.sh": true,
				"proxy_http='http://proxy.local' bash ./install.sh -v ":                        true,
				"proxy_http='http://proxy.local' rm -f install.sh":                             true,
			},
		},

		"NOProxy": {
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
				"proxy_http='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"curl -LO https://www.chef.io/chef/install.sh": true,
				"proxy_http='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"bash ./install.sh -v ": true,
				"proxy_http='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
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
				"bash ./install.sh -v 11.18.6":                 true,
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

		err = p.sshInstallChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestResourceProvider_sshCreateConfigFiles(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Sudo": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"sudo mkdir -p " + linuxConfDir:           true,
				"sudo chmod 777 " + linuxConfDir:          true,
				"sudo chmod 755 " + linuxConfDir:          true,
				"sudo chown -R root.root " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				"/etc/chef/validation.pem":  "VALIDATOR-PEM-FILE",
				"/etc/chef/client.rb":       defaultSSHClientConf,
				"/etc/chef/first-boot.json": `{"run_list":["cookbook::recipe"]}`,
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

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				"/etc/chef/validation.pem":  "VALIDATOR-PEM-FILE",
				"/etc/chef/client.rb":       defaultSSHClientConf,
				"/etc/chef/first-boot.json": `{"run_list":["cookbook::recipe"]}`,
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
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				"/etc/chef/validation.pem":  "VALIDATOR-PEM-FILE",
				"/etc/chef/client.rb":       proxySSHClientConf,
				"/etc/chef/first-boot.json": `{"run_list":["cookbook::recipe"]}`,
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
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				"/etc/chef/validation.pem": "VALIDATOR-PEM-FILE",
				"/etc/chef/client.rb":      defaultSSHClientConf,
				"/etc/chef/first-boot.json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
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

		err = p.sshCreateConfigFiles(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

const defaultSSHClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local"
validation_client_name  "validator"
node_name               "nodename1"`

const proxySSHClientConf = `log_location            STDOUT
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
