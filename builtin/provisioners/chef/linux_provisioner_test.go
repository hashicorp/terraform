package chef

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/mitchellh/go-linereader"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_linuxInstallChefClient(t *testing.T) {
	cases := map[string]createConfigTestCase{
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
				"rm -f install.sh":                              true,
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
	cases := map[string]createConfigTestCase{
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
				"sudo chown -R root.root " + path.Join(linuxConfDir, "ohai/hints"):       true,
				"sudo chmod 755 " + linuxConfDir:                                         true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 600):                          true,
				"sudo chown -R root.root " + linuxConfDir:                                true,
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

		"LocalMode": {
			Config: map[string]interface{}{
				"use_local_mode": true,
				"chef_repo":      "tbd",
				"run_list":       []interface{}{"role[testrole]"},
				"environment":    "testenv",
				"node_name":      "testnode",
			},

			Commands: map[string]bool{
				// prepare conf dir for upload
				"sudo mkdir -p " + linuxConfDir:                 true,
				"sudo chmod 777 " + linuxConfDir:                true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 666): true,
				// restore conf dir
				"sudo chmod 755 " + linuxConfDir:                true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 600): true,
				"sudo chown -R root.root " + linuxConfDir:       true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":       linuxLocalModeClientConf,
				linuxConfDir + "/first-boot.json": `{"run_list":["role[testrole]"]}`,
			},
		},

		"LocalModePolicy": {
			Config: map[string]interface{}{
				"use_local_mode": true,
				"chef_repo":      "tbd",
				"use_policyfile": true,
				"policy_name":    "testpolicy",
			},

			Commands: map[string]bool{
				// prepare conf dir for upload
				"sudo mkdir -p " + linuxConfDir:                 true,
				"sudo chmod 777 " + linuxConfDir:                true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 666): true,
				// restore conf dir
				"sudo chmod 755 " + linuxConfDir:                true,
				"sudo " + fmt.Sprintf(chmod, linuxConfDir, 600): true,
				"sudo chown -R root.root " + linuxConfDir:       true,
			},

			Uploads: map[string]string{
				linuxConfDir + "/client.rb":       linuxLocalModePolicyClientConf,
				linuxConfDir + "/first-boot.json": `{}`,
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

func TestResourceProvider_linuxDeployChefRepo(t *testing.T) {
	cases := map[string]createConfigTestCase{
		"LocalMode": {
			Config: map[string]interface{}{
				"use_local_mode": true,
				"chef_repo":      "tbd",
				"run_list":       []interface{}{"role[testrole]"},
				"environment":    "testenv",
			},

			Commands: map[string]bool{
				// prepare var dir for upload
				"sudo mkdir -p " + linuxRepoDir:                 true,
				"sudo chmod 777 " + linuxRepoDir:                true,
				"sudo " + fmt.Sprintf(chmod, linuxRepoDir, 666): true,
				// client run
				"sudo sh -c 'cd /var/chef && chef-client -z -j /etc/chef/first-boot.json -E testenv'": true,
				"sudo chef-client -z -j /etc/chef/first-boot.json -E testenv":                         false,
			},

			UploadDirs: map[string]string{},

			CheckLog: func(log string) {
				found := false
				lr := linereader.New(strings.NewReader(log))
				for line := range lr.Ch {
					if strings.Contains(line, "[WARN] Chef repository lacks directory 'roles'. This is probably a mistake.") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected warning about missing 'roles' directory")
				}
			},

			SetUp: func(t *testing.T, self createConfigTestCase) error {
				layout := map[string][]string{
					"cookbooks":    nil,
					"environments": nil,
				}

				path, err := testCreateTmpFiles(layout)
				if err != nil {
					return err
				}

				self.Config["chef_repo"] = path
				self.UploadDirs[path+"/"] = linuxRepoDir
				return nil
			},

			TearDown: func(t *testing.T, self createConfigTestCase) error {
				return testDeleteTmpFiles(self.Config["chef_repo"].(string))
			},
		},

		"LocalModePolicy": {
			Config: map[string]interface{}{
				"use_local_mode": true,
				"chef_repo":      "tbd",
				"use_policyfile": true,
				"policy_name":    "testpolicy",
			},

			Commands: map[string]bool{
				// prepare var dir for upload
				"sudo mkdir -p " + linuxRepoDir:                 true,
				"sudo chmod 777 " + linuxRepoDir:                true,
				"sudo " + fmt.Sprintf(chmod, linuxRepoDir, 666): true,
				// client run
				"sudo sh -c 'cd /var/chef && chef-client -z -j /etc/chef/first-boot.json'": true,
				"sudo chef-client -z -j /etc/chef/first-boot.json":                         false,
			},

			UploadDirs: map[string]string{},

			CheckLog: func(log string) {
				found := false
				lr := linereader.New(strings.NewReader(log))
				for line := range lr.Ch {
					if strings.Contains(line, "[WARN] Chef repository lacks directory 'cookbook_artifacts'. This is probably a mistake.") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected warning about missing 'cookbook_artifacts' directory")
				}
			},

			SetUp: func(t *testing.T, self createConfigTestCase) error {
				layout := map[string][]string{
					"": []string{"Policyfile.lock.json"},
				}
				path, err := testCreateTmpFiles(layout)
				if err != nil {
					return err
				}

				self.Config["chef_repo"] = path
				self.UploadDirs[path+"/"] = linuxRepoDir
				return nil
			},

			TearDown: func(t *testing.T, self createConfigTestCase) error {
				return testDeleteTmpFiles(self.Config["chef_repo"].(string))
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads
		c.UploadDirs = tc.UploadDirs

		if tc.SetUp != nil {
			if err := tc.SetUp(t, tc); err != nil {
				if tderr := tc.TearDown(t, tc); tderr != nil {
					t.Logf("TearDown failed for %s: %v", k, err)
				}
				t.Fatalf("SetUp failed for %s: %v", k, err)
			}

			defer tc.TearDown(t, tc)
		}

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.useSudo = !p.PreventSudo

		var logBuf bytes.Buffer
		logWriter := bufio.NewWriter(&logBuf)
		log.SetOutput(logWriter)

		err = p.linuxUploadChefRepo(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}

		logWriter.Flush()
		logs := logBuf.String()
		os.Stderr.WriteString(logs)
		if tc.CheckLog != nil {
			tc.CheckLog(logs)
		}
	}
}

const defaultLinuxClientConf = `log_location            STDOUT
node_name               "nodename1"
chef_server_url         "https://chef.local/"`

const proxyLinuxClientConf = `log_location            STDOUT
node_name               "nodename1"
chef_server_url         "https://chef.local/"

http_proxy          "http://proxy.local"
ENV['http_proxy'] = "http://proxy.local"
ENV['HTTP_PROXY'] = "http://proxy.local"

https_proxy          "https://proxy.local"
ENV['https_proxy'] = "https://proxy.local"
ENV['HTTPS_PROXY'] = "https://proxy.local"

no_proxy          "http://local.local,https://local.local"
ENV['no_proxy'] = "http://local.local,https://local.local"

ssl_verify_mode  :verify_none`

const linuxLocalModeClientConf = `log_location            STDOUT
node_name               "testnode"
chef_repo_path          "/tmp/chef-repo"`

const linuxLocalModePolicyClientConf = `log_location            STDOUT
chef_repo_path          "/tmp/chef-repo"

use_policyfile true
policy_group 	 "local"
policy_name 	 "testpolicy"`
