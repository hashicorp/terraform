package chef

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

func TestResourceProvider_linuxInstallChefClient(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
	}{
		"Sudo": {
			Config: map[string]interface{}{
				"node_name":   "nodename1",
				"run_list":    []interface{}{"cookbook::recipe"},
				"user_name":   "bob",
				"instance_id": "myid",
				"user_key":    "USER-KEY",
			},

			Commands: map[string]bool{
				"sudo bash -c 'curl -LO https://omnitruck.chef.io/install.sh'": true,
				"sudo bash -c 'bash ./install.sh -v \"\" -c stable'":           true,
				"sudo bash -c 'rm -f install.sh'":                              true,
			},
		},

		"NoSudo": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"secret_key":   "SECRET-KEY",

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
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

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
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

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
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

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
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

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
				"version":  "11.18.6",
			},

			Commands: map[string]bool{
				"curl -LO https://omnitruck.chef.io/install.sh": true,
				"bash ./install.sh -v \"11.18.6\" -c stable":    true,
				"rm -f install.sh":                              true,
			},
		},

		"Channel": {
			Config: map[string]interface{}{
				"channel":      "current",
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},

				"user_name":   "bob",
				"instance_id": "myid",

				"user_key": "USER-KEY",
				"version":  "11.18.6",
			},

			Commands: map[string]bool{
				"curl -LO https://omnitruck.chef.io/install.sh": true,
				"bash ./install.sh -v \"11.18.6\" -c current":   true,
				"rm -f install.sh":                              true,
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
		Config     map[string]interface{}
		Commands   map[string]bool
		Uploads    map[string]string
		UploadDirs map[string]string
	}{
		"Sudo": {
			Config: map[string]interface{}{
				"ohai_hints": []interface{}{"test-fixtures/ohaihint.json"},
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"secret_key": "SECRET-KEY",

				"user_name":       "bob",
				"dir_resources":   "test-fixtures",
				"local_nodes_dir": "nodes",
				"instance_id":     "myid",
				"user_key":        "USER-KEY",
			},

			Commands: map[string]bool{
				"sudo bash -c 'mkdir -p " + linuxConfDir + "'":                 true,
				"sudo bash -c 'chmod 777 " + linuxConfDir + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, linuxConfDir, 666) + "'": true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "ohai/hints") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "ohai/hints") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "ohai/hints"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "ohai/hints") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "ohai/hints"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "ohai/hints") + "'":       true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "nodes") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "nodes") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "nodes"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "nodes") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "nodes"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "nodes") + "'":       true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "data_bags") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "data_bags") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "data_bags"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "data_bags") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "data_bags"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "data_bags") + "'":       true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "cookbooks") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "cookbooks") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "cookbooks"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "cookbooks") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "cookbooks"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "cookbooks") + "'":       true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "dna") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "dna") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "dna"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "dna") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "dna"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "dna") + "'":       true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "roles") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "roles") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "roles"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "roles") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "roles"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "roles") + "'":       true,

				"sudo bash -c 'mkdir -p " + path.Join(linuxConfDir, "environments") + "'":                 true,
				"sudo bash -c 'chmod 777 " + path.Join(linuxConfDir, "environments") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "environments"), 666) + "'": true,
				"sudo bash -c 'chmod 755 " + path.Join(linuxConfDir, "environments") + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, path.Join(linuxConfDir, "environments"), 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + path.Join(linuxConfDir, "environments") + "'":       true,

				"sudo bash -c 'chmod 755 " + linuxConfDir + "'":                true,
				"sudo bash -c '" + fmt.Sprintf(chmod, linuxConfDir, 600) + "'": true,
				"sudo bash -c 'chown -R root.root " + linuxConfDir + "'":       true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/dna/myid.json":             `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/ohai/hints/ohaihint.json":  "OHAI-HINT-FILE",
				linuxConfDir + "/bob.pem":                   "USER-KEY",
			},

			UploadDirs: map[string]string{
				"test-fixtures/data_bags":    linuxConfDir,
				"test-fixtures/dna":          linuxConfDir,
				"test-fixtures/cookbooks":    linuxConfDir,
				"test-fixtures/environments": linuxConfDir,
				"test-fixtures/nodes":        linuxConfDir,
				"test-fixtures/roles":        linuxConfDir,
			},
		},

		"NoSudo": {
			Config: map[string]interface{}{
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"secret_key":   "SECRET-KEY",

				"user_name":       "bob",
				"instance_id":     "myid",
				"dir_resources":   "test-fixtures",
				"local_nodes_dir": "nodes",
				"user_key":        "USER-KEY",
			},

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir:                   true,
				"mkdir -p " + linuxConfDir + "/data_bags":    true,
				"mkdir -p " + linuxConfDir + "/cookbooks":    true,
				"mkdir -p " + linuxConfDir + "/nodes":        true,
				"mkdir -p " + linuxConfDir + "/roles":        true,
				"mkdir -p " + linuxConfDir + "/dna":          true,
				"mkdir -p " + linuxConfDir + "/environments": true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/dna/myid.json":             `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/bob.pem":                   "USER-KEY",
			},

			UploadDirs: map[string]string{
				"test-fixtures/data_bags":    linuxConfDir,
				"test-fixtures/dna":          linuxConfDir,
				"test-fixtures/cookbooks":    linuxConfDir,
				"test-fixtures/environments": linuxConfDir,
				"test-fixtures/nodes":        linuxConfDir,
				"test-fixtures/roles":        linuxConfDir,
			},
		},

		"Proxy": {
			Config: map[string]interface{}{
				"http_proxy":   "http://proxy.local",
				"https_proxy":  "https://proxy.local",
				"no_proxy":     []interface{}{"http://local.local", "https://local.local"},
				"node_name":    "nodename1",
				"prevent_sudo": true,
				"run_list":     []interface{}{"cookbook::recipe"},
				"secret_key":   "SECRET-KEY",

				"ssl_verify_mode": "verify_none",
				"user_name":       "bob",
				"instance_id":     "myid",
				"dir_resources":   "test-fixtures",
				"local_nodes_dir": "nodes",
				"user_key":        "USER-KEY",
			},

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir:                   true,
				"mkdir -p " + linuxConfDir + "/data_bags":    true,
				"mkdir -p " + linuxConfDir + "/cookbooks":    true,
				"mkdir -p " + linuxConfDir + "/nodes":        true,
				"mkdir -p " + linuxConfDir + "/roles":        true,
				"mkdir -p " + linuxConfDir + "/dna":          true,
				"mkdir -p " + linuxConfDir + "/environments": true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 proxyLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/dna/myid.json":             `{"run_list":["cookbook::recipe"]}`,
				linuxConfDir + "/bob.pem":                   "USER-KEY",
			},

			UploadDirs: map[string]string{
				"test-fixtures/data_bags":    linuxConfDir,
				"test-fixtures/dna":          linuxConfDir,
				"test-fixtures/cookbooks":    linuxConfDir,
				"test-fixtures/environments": linuxConfDir,
				"test-fixtures/nodes":        linuxConfDir,
				"test-fixtures/roles":        linuxConfDir,
			},
		},

		"DNAAttributes": {
			Config: map[string]interface{}{
				"dna_attributes": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3", "id" : "1"}}},"key2":"value2","ipaddress" : "0.0.0.0"}`,
				"automatic_attributes": `{"test":{"subkey1" : "value"} }`,
				"default_attributes":   `{"test_default":{"subkey_default" : "value"} }`,
				"node_name":            "nodename1",
				"prevent_sudo":         true,
				"run_list":             []interface{}{"cookbook::recipe"},
				"secret_key":           "SECRET-KEY",

				"user_name":       "bob",
				"instance_id":     "myid",
				"dir_resources":   "test-fixtures",
				"local_nodes_dir": "nodes",
				"user_key":        "USER-KEY",
			},

			Commands: map[string]bool{
				"mkdir -p " + linuxConfDir:                   true,
				"mkdir -p " + linuxConfDir + "/data_bags":    true,
				"mkdir -p " + linuxConfDir + "/cookbooks":    true,
				"mkdir -p " + linuxConfDir + "/nodes":        true,
				"mkdir -p " + linuxConfDir + "/roles":        true,
				"mkdir -p " + linuxConfDir + "/dna":          true,
				"mkdir -p " + linuxConfDir + "/environments": true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":                 defaultLinuxClientConf,
				linuxConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				linuxConfDir + "/bob.pem":                   "USER-KEY",
				linuxConfDir + "/dna/myid.json": `{"ipaddress":"192.168.0.1","key1":{"subkey1":{"subkey2a":` +
					`["val1","val2","val3"],"subkey2b":{"id":"1","subkey3":"value3"}}},"key2":"value2","run_list":["cookbook::recipe"]}`,
			},

			UploadDirs: map[string]string{
				"test-fixtures/data_bags":    linuxConfDir,
				"test-fixtures/dna":          linuxConfDir,
				"test-fixtures/cookbooks":    linuxConfDir,
				"test-fixtures/environments": linuxConfDir,
				"test-fixtures/nodes":        linuxConfDir,
				"test-fixtures/roles":        linuxConfDir,
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads
		c.UploadDirs = tc.UploadDirs

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		p.DefaultConfDir = linuxConfDir

		p.useSudo = !p.PreventSudo

		if err = p.linuxCreateConfigFiles(o, c); err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
		os.Remove("test-fixtures/nodes/myid.json")
	}
}

const defaultLinuxClientConf = `log_location            STDOUT


local_mode true

cookbook_path '/opt/chef/0/cookbooks'

node_path '/opt/chef/0/nodes'
role_path '/opt/chef/0/roles'
data_bag_path '/opt/chef/0/data_bags'
rubygems_url 'http://nexus.query.consul/content/groups/rubygems'
environment_path '/opt/chef/0/environments'`

const proxyLinuxClientConf = `log_location            STDOUT

http_proxy          "http://proxy.local"
ENV['http_proxy'] = "http://proxy.local"
ENV['HTTP_PROXY'] = "http://proxy.local"

https_proxy          "https://proxy.local"
ENV['https_proxy'] = "https://proxy.local"
ENV['HTTPS_PROXY'] = "https://proxy.local"

no_proxy          "http://local.local,https://local.local"
ENV['no_proxy'] = "http://local.local,https://local.local"

ssl_verify_mode  :verify_none

local_mode true

cookbook_path '/opt/chef/0/cookbooks'

node_path '/opt/chef/0/nodes'
role_path '/opt/chef/0/roles'
data_bag_path '/opt/chef/0/data_bags'
rubygems_url 'http://nexus.query.consul/content/groups/rubygems'
environment_path '/opt/chef/0/environments'`
