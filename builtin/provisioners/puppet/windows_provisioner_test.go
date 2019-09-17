package puppet

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	getHostByNameCmd     = `powershell -Command "& {([System.Net.Dns]::GetHostByName(($env:computerName))).Hostname}"`
	domainQueryCmd       = `powershell -Command "& {(Get-WmiObject -Query 'select DNSDomain from Win32_NetworkAdapterConfiguration where IPEnabled = True').DNSDomain}"`
	downloadInstallerCmd = `powershell -Command "& {[Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}; (New-Object System.Net.WebClient).DownloadFile('https://puppet.test.com:8140/packages/current/install.ps1', 'install.ps1')}"`
	runInstallerCmd      = `powershell -Command "& .\install.ps1 -PuppetServiceEnsure stopped"`
	runPuppetCmd         = "puppet agent --test --server puppet.test.com --environment production"
)

func TestResourceProvisioner_windowsUploadFile(t *testing.T) {
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
				`powershell.exe new-item -itemtype directory -force -path C:\ProgramData\PuppetLabs\puppet\etc`: true,
			},
			Uploads: map[string]string{
				`C:\ProgramData\PuppetLabs\puppet\etc\csr_attributes.yaml`: "",
			},
			Dir:      `C:\ProgramData\PuppetLabs\puppet\etc`,
			Filename: "csr_attributes.yaml",
			File:     strings.NewReader(""),
		},
		"Failure when creating the directory": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			Commands: map[string]bool{
				`powershell.exe new-item -itemtype directory -force -path C:\ProgramData\PuppetLabs\puppet\etc`: true,
			},
			Dir:      `C:\ProgramData\PuppetLabs\puppet\etc`,
			Filename: "csr_attributes.yaml",
			File:     strings.NewReader(""),
			CommandFunc: func(r *remote.Cmd) error {
				r.SetExitStatus(1, &remote.ExitError{
					Command:    `powershell.exe new-item -itemtype directory -force -path C:\ProgramData\PuppetLabs\puppet\etc`,
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

		err = p.windowsUploadFile(tc.File, tc.Dir, tc.Filename)
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

func TestResourceProvisioner_windowsDefaultCertname(t *testing.T) {
	cases := map[string]struct {
		Config        map[string]interface{}
		Commands      map[string]bool
		CommandFunc   func(*remote.Cmd) error
		ExpectedError bool
	}{
		"GetHostByName failure": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				switch r.Command {
				case getHostByNameCmd:
					r.SetExitStatus(1, &remote.ExitError{
						Command:    getHostByNameCmd,
						ExitStatus: 1,
						Err:        nil,
					})
				default:
					return fmt.Errorf("Command not found!")
				}

				return nil
			},
			ExpectedError: true,
		},
		"GetHostByName returns FQDN": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				switch r.Command {
				case getHostByNameCmd:
					r.Stdout.Write([]byte("example.test.com\n"))
					time.Sleep(200 * time.Millisecond)
					r.SetExitStatus(0, nil)
				default:
					return fmt.Errorf("Command not found!")
				}

				return nil
			},
		},
		"GetHostByName returns hostname, DNSDomain query succeeds": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				switch r.Command {
				case getHostByNameCmd:
					r.Stdout.Write([]byte("example\n"))
					time.Sleep(200 * time.Millisecond)
					r.SetExitStatus(0, nil)
				case domainQueryCmd:
					r.Stdout.Write([]byte("test.com\n"))
					time.Sleep(200 * time.Millisecond)
					r.SetExitStatus(0, nil)
				default:
					return fmt.Errorf("Command not found!")
				}

				return nil
			},
		},
		"GetHostByName returns hostname, DNSDomain query fails": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				switch r.Command {
				case getHostByNameCmd:
					r.Stdout.Write([]byte("example\n"))
					time.Sleep(200 * time.Millisecond)
					r.SetExitStatus(0, nil)
				case domainQueryCmd:
					r.SetExitStatus(1, &remote.ExitError{
						Command:    domainQueryCmd,
						ExitStatus: 1,
						Err:        nil,
					})
				default:
					return fmt.Errorf("Command not found!")
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

		_, err = p.windowsDefaultCertname()
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

func TestResourceProvisioner_windowsInstallPuppetAgent(t *testing.T) {
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
				downloadInstallerCmd: true,
				runInstallerCmd:      true,
			},
		},
		"Installer download fails": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": true,
			},
			CommandFunc: func(r *remote.Cmd) error {
				switch r.Command {
				case downloadInstallerCmd:
					r.SetExitStatus(1, &remote.ExitError{
						Command:    downloadInstallerCmd,
						ExitStatus: 1,
						Err:        nil,
					})
				default:
					return fmt.Errorf("Command not found!")
				}

				return nil
			},
			ExpectedError: true,
		},
		"Install script fails": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				switch r.Command {
				case downloadInstallerCmd:
					r.SetExitStatus(0, nil)
				case runInstallerCmd:
					r.SetExitStatus(1, &remote.ExitError{
						Command:    runInstallerCmd,
						ExitStatus: 1,
						Err:        nil,
					})
				default:
					return fmt.Errorf("Command not found!")
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

		err = p.windowsInstallPuppetAgent()
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

func TestResourceProvisioner_windowsRunPuppetAgent(t *testing.T) {
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
				runPuppetCmd: true,
			},
		},
		"When puppet returns 2 (changes applied without error)": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				r.SetExitStatus(2, &remote.ExitError{
					Command:    runPuppetCmd,
					ExitStatus: 2,
					Err:        nil,
				})
				return nil
			},
		},
		"When puppet returns something not 0 or 2": {
			Config: map[string]interface{}{
				"server":   "puppet.test.com",
				"use_sudo": false,
			},
			CommandFunc: func(r *remote.Cmd) error {
				r.SetExitStatus(1, &remote.ExitError{
					Command:    runPuppetCmd,
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

		err = p.windowsRunPuppetAgent()
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
