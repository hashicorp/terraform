package chef

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_linuxInstallChefClient(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
	}{
		"Sudo": {
			Config: map[string]interface{}{
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
			},

			Commands: map[string]bool{
				"sudo curl -LO https://omnitruck.chef.io/install.sh": true,
				"sudo bash ./install.sh -v \"\" -c stable":           true,
				"sudo rm -f install.sh":                              true,
			},
		},

		"NoSudo": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"secret_key":   "SECRET-KEY",
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
			},

			Commands: map[string]bool{
				"curl -LO https://omnitruck.chef.io/install.sh": true,
				"bash ./install.sh -v \"\" -c stable":           true,
				"rm -f install.sh":                              true,
			},
		},

		"HTTPProxy": {
			Config: map[string]interface{}{
				"http_proxy":   "http://proxy.local",
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
			},

			Commands: map[string]bool{
				"http_proxy='http://proxy.local' curl -LO https://omnitruck.chef.io/install.sh": true,
				"http_proxy='http://proxy.local' bash ./install.sh -v \"\" -c stable":           true,
				"http_proxy='http://proxy.local' rm -f install.sh":                              true,
			},
		},

		"HTTPSProxy": {
			Config: map[string]interface{}{
				"https_proxy":  "https://proxy.local",
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
			},

			Commands: map[string]bool{
				"https_proxy='https://proxy.local' curl -LO https://omnitruck.chef.io/install.sh": true,
				"https_proxy='https://proxy.local' bash ./install.sh -v \"\" -c stable":           true,
				"https_proxy='https://proxy.local' rm -f install.sh":                              true,
			},
		},

		"NoProxy": {
			Config: map[string]interface{}{
				"http_proxy":   "http://proxy.local",
				"no_proxy":     []interface{}{"http://local.local", "http://local.org"},
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
			},

			Commands: map[string]bool{
				"http_proxy='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"curl -LO https://omnitruck.chef.io/install.sh": true,
				"http_proxy='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"bash ./install.sh -v \"\" -c stable": true,
				"http_proxy='http://proxy.local' no_proxy='http://local.local,http://local.org' " +
					"rm -f install.sh": true,
			},
		},

		"Version": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
				"version":      "11.18.6",
			},

			Commands: map[string]bool{
				"curl -LO https://omnitruck.chef.io/install.sh": true,
				"bash ./install.sh -v \"11.18.6\" -c stable":    true,
				"rm -f install.sh": true,
			},
		},

		"Channel": {
			Config: map[string]interface{}{
				"channel":      "current",
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
				"version":      "11.18.6",
			},

			Commands: map[string]bool{
				"curl -LO https://omnitruck.chef.io/install.sh": true,
				"bash ./install.sh -v \"11.18.6\" -c current":   true,
				"rm -f install.sh": true,
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

		p.useSudo = !p.PreventSudo

		err = p.linuxInstallChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestResourceProvider_linuxCreateConfigFiles(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Sudo": {
			Config: map[string]interface{}{
				"ohai_hints": []interface{}{"test-fixtures/ohaihint.json"},
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"secret_key": "SECRET-KEY",
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
			},

			Commands: map[string]bool{
				"sudo mkdir -p " + linuxConfDir:                                          true,
				"sudo chmod 777 " + linuxConfDir:                                         true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 666):                          true,
				"sudo mkdir -p " + path.Join(linuxConfDir, "ohai/hints"):                 true,
				"sudo chmod 777 " + path.Join(linuxConfDir, "ohai/hints"):                true,
				"sudo " + fmt.Sprintf(chmod, path.Join(linuxConfDir, "ohai/hints"), 666): true,
				"sudo chmod 755 " + path.Join(linuxConfDir, "ohai/hints"):                true,
				"sudo " + fmt.Sprintf(chmod, path.Join(linuxConfDir, "ohai/hints"), 600): true,
				"sudo chown -R root:root " + path.Join(linuxConfDir, "ohai/hints"):       true,
				"sudo chmod 755 " + linuxConfDir:                                         true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 600):                          true,
				"sudo chown -R root:root " + linuxConfDir:                                true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/ohai/hints/ohaihint.json":  "OHAI-HINT-FILE",
				linuxConfDir + "/bob.pem":                   "USER-KEY",
			},
		},

		"NoSudo": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"secret_key":   "SECRET-KEY",
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
			},

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/bob.pem":                   "USER-KEY",
			},
		},

		"Proxy": {
			Config: map[string]interface{}{
				"http_proxy":      "http://proxy.local",
				"https_proxy":     "https://proxy.local",
				"no_proxy":        []interface{}{"http://local.local", "https://local.local"},
				"node_name":       "nodename1",
				"prevent_sudo":    true,
				"run_list":        []interface{}{"cookbook::recipe"},
				"secret_key":      "SECRET-KEY",
				"server_url":      "https://chef.local",
				"ssl_verify_mode": "verify_none",
				"user_name":       "bob",
				"user_key":        "USER-KEY",
			},

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 proxyLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/bob.pem":                   "USER-KEY",
			},
		},

		"Attributes JSON": {
			Config: map[string]interface{}{
				"attributes_json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3"}}},"key2":"value2"}`,
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"secret_key":   "SECRET-KEY",
				"server_url":   "https://chef.local",
				"user_name":    "bob",
				"user_key":     "USER-KEY",
			},

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir: true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/bob.pem":                   "USER-KEY",
				linuxConfDir + "/first-boot.json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3"}}},"key2":"value2","run_list":["cookbook::recipe"]}`,
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
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
chef_server_url         "https://chef.local/"
node_name               "nodename1"`

const proxyLinuxClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local/"
node_name               "nodename1"

http_proxy          "http://proxy.local"
ENV['http_proxy'] = "http://proxy.local"
ENV['HTTP_PROXY'] = "http://proxy.local"

https_proxy          "https://proxy.local"
ENV['https_proxy'] = "https://proxy.local"
ENV['HTTPS_PROXY'] = "https://proxy.local"

no_proxy          "http://local.local,https://local.local"
ENV['no_proxy'] = "http://local.local,https://local.local"

ssl_verify_mode  :verify_none`
