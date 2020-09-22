package puppet

import (
	"io"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_linuxUploadFile(t *testing.T) {
	cases := map[string]struct {
		Config        map[string]interface{}
		Commands      map[string]bool
		CommandFunc   func(*remote.Cmd) error
		ExpectedError bool
		Uploads       map[string]string
		File          io.Reader
		Dir           string
		Filename      string
	}{
		"Successful upload": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"mkdir -p /etc/puppetlabs/puppet":                                        true,
				"mv /tmp/csr_attributes.yaml /etc/puppetlabs/puppet/csr_attributes.yaml": true,
			},
			Uploads: map[string]string{
				"/tmp/csr_attributes.yaml": "",
			},
			Dir:      "/etc/puppetlabs/puppet",
			Filename: "csr_attributes.yaml",
			File:     strings.NewReader(""),
		},
		"Failure when creating the directory": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"mkdir -p /etc/puppetlabs/puppet": true,
			},
			Dir:      "/etc/puppetlabs/puppet",
			Filename: "csr_attributes.yaml",
			File:     strings.NewReader(""),
			CommandFunc: func(r *remote.Cmd) error {
				r.SetExitStatus(1, &remote.ExitError{
					Command:    "mkdir -p /etc/puppetlabs/puppet",
					ExitStatus: 1,
					Err:        nil,
				})
				return nil
			},
			ExpectedError: true,
		},
	}

	for k, tc := range cases {
		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		c := new(communicator.MockCommunicator)
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads
		if tc.CommandFunc != nil {
			c.CommandFunc = tc.CommandFunc
		}
		p.comm = c
		p.output = new(terraform.MockUIOutput)

		err = p.linuxUploadFile(tc.File, tc.Dir, tc.Filename)
		if tc.ExpectedError {
			if err == nil {
				t.Fatalf("Expected error, but no error returned")
			}
		} else {
			if err != nil {
				t.Fatalf("Test %q failed: %v", k, err)
			}
		}
	}
}

func TestResourceProvisioner_linuxDefaultCertname(t *testing.T) {
	cases := map[string]struct {
		Config        map[string]interface{}
		Commands      map[string]bool
		CommandFunc   func(*remote.Cmd) error
		ExpectedError bool
	}{
		"No sudo": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"hostname -f": true,
			},
		},
		"With sudo": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": true,
			},
			Commands: map[string]bool{
				"sudo hostname -f": true,
			},
		},
		"Failed execution": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"hostname -f": true,
			},
			CommandFunc: func(r *remote.Cmd) error {
				if r.Command == "hostname -f" {
					r.SetExitStatus(1, &remote.ExitError{
						Command:    "hostname -f",
						ExitStatus: 1,
						Err:        nil,
					})
				} else {
					r.SetExitStatus(0, nil)
				}
				return nil
			},
			ExpectedError: true,
		},
	}

	for k, tc := range cases {
		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		c := new(communicator.MockCommunicator)
		c.Commands = tc.Commands
		if tc.CommandFunc != nil {
			c.CommandFunc = tc.CommandFunc
		}
		p.comm = c
		p.output = new(terraform.MockUIOutput)

		_, err = p.linuxDefaultCertname()
		if tc.ExpectedError {
			if err == nil {
				t.Fatalf("Expected error, but no error returned")
			}
		} else {
			if err != nil {
				t.Fatalf("Test %q failed: %v", k, err)
			}
		}
	}
}

func TestResourceProvisioner_linuxInstallPuppetAgent(t *testing.T) {
	cases := map[string]struct {
		Config        map[string]interface{}
		Commands      map[string]bool
		CommandFunc   func(*remote.Cmd) error
		ExpectedError bool
	}{
		"Everything runs succcessfully": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"curl -kO https://puppet.test.com:8140/packages/current/install.bash": true,
				"bash -- ./install.bash --puppet-service-ensure stopped":              true,
				"rm -f install.bash": true,
			},
		},
		"Respects the use_sudo config flag": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": true,
			},
			Commands: map[string]bool{
				"sudo curl -kO https://puppet.test.com:8140/packages/current/install.bash": true,
				"sudo bash -- ./install.bash --puppet-service-ensure stopped":              true,
				"sudo rm -f install.bash": true,
			},
		},
		"When the curl command fails": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"curl -kO https://puppet.test.com:8140/packages/current/install.bash": true,
				"bash -- ./install.bash --puppet-service-ensure stopped":              false,
				"rm -f install.bash": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				if r.Command == "curl -kO https://puppet.test.com:8140/packages/current/install.bash" {
					r.SetExitStatus(1, &remote.ExitError{
						Command:    "curl -kO https://puppet.test.com:8140/packages/current/install.bash",
						ExitStatus: 1,
						Err:        nil,
					})
				} else {
					r.SetExitStatus(0, nil)
				}
				return nil
			},
			ExpectedError: true,
		},
		"When the install script fails": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"curl -kO https://puppet.test.com:8140/packages/current/install.bash": true,
				"bash -- ./install.bash --puppet-service-ensure stopped":              true,
				"rm -f install.bash": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				if r.Command == "bash -- ./install.bash --puppet-service-ensure stopped" {
					r.SetExitStatus(1, &remote.ExitError{
						Command:    "bash -- ./install.bash --puppet-service-ensure stopped",
						ExitStatus: 1,
						Err:        nil,
					})
				} else {
					r.SetExitStatus(0, nil)
				}
				return nil
			},
			ExpectedError: true,
		},
		"When the cleanup rm fails": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"curl -kO https://puppet.test.com:8140/packages/current/install.bash": true,
				"bash -- ./install.bash --puppet-service-ensure stopped":              true,
				"rm -f install.bash": true,
			},
			CommandFunc: func(r *remote.Cmd) error {
				if r.Command == "rm -f install.bash" {
					r.SetExitStatus(1, &remote.ExitError{
						Command:    "rm -f install.bash",
						ExitStatus: 1,
						Err:        nil,
					})
				} else {
					r.SetExitStatus(0, nil)
				}
				return nil
			},
			ExpectedError: true,
		},
	}

	for k, tc := range cases {
		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		c := new(communicator.MockCommunicator)
		c.Commands = tc.Commands
		if tc.CommandFunc != nil {
			c.CommandFunc = tc.CommandFunc
		}
		p.comm = c
		p.output = new(terraform.MockUIOutput)

		err = p.linuxInstallPuppetAgent()
		if tc.ExpectedError {
			if err == nil {
				t.Fatalf("Expected error, but no error returned")
			}
		} else {
			if err != nil {
				t.Fatalf("Test %q failed: %v", k, err)
			}
		}
	}
}

func TestResourceProvisioner_linuxRunPuppetAgent(t *testing.T) {
	cases := map[string]struct {
		Config        map[string]interface{}
		Commands      map[string]bool
		CommandFunc   func(*remote.Cmd) error
		ExpectedError bool
	}{
		"When puppet returns 0": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				"/opt/puppetlabs/puppet/bin/puppet agent --test --server puppet.test.com --environment production": true,
			},
		},
		"When puppet returns 2 (changes applied without error)": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				r.SetExitStatus(2, &remote.ExitError{
					Command:    "/opt/puppetlabs/puppet/bin/puppet agent --test --server puppet.test.com",
					ExitStatus: 2,
					Err:        nil,
				})
				return nil
			},
			ExpectedError: false,
		},
		"When puppet returns something not 0 or 2": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				r.SetExitStatus(1, &remote.ExitError{
					Command:    "/opt/puppetlabs/puppet/bin/puppet agent --test --server puppet.test.com",
					ExitStatus: 1,
					Err:        nil,
				})
				return nil
			},
			ExpectedError: true,
		},
	}

	for k, tc := range cases {
		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		c := new(communicator.MockCommunicator)
		c.Commands = tc.Commands
		if tc.CommandFunc != nil {
			c.CommandFunc = tc.CommandFunc
		}
		p.comm = c
		p.output = new(terraform.MockUIOutput)

		err = p.linuxRunPuppetAgent()
		if tc.ExpectedError {
			if err == nil {
				t.Fatalf("Expected error, but no error returned")
			}
		} else {
			if err != nil {
				t.Fatalf("Test %q failed: %v", k, err)
			}
		}
	}
}
