package chefclient

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

func (p *Provisioner) sshInstallChefClient(
	o terraform.UIOutput,
	comm communicator.Communicator) error {
	var installCmd bytes.Buffer

	// Build up a single command based on the given config options
	installCmd.WriteString("curl")
	if p.HTTPProxy != "" {
		installCmd.WriteString(" --proxy " + p.HTTPProxy)
	}
	if p.NOProxy != nil {
		installCmd.WriteString(" --noproxy " + strings.Join(p.NOProxy, ","))
	}
	installCmd.WriteString(" -LO https://www.chef.io/chef/install.sh 2>/dev/null &&")
	if !p.PreventSudo {
		installCmd.WriteString(" sudo")
	}
	installCmd.WriteString(" bash ./install.sh")
	if p.Version != "" {
		installCmd.WriteString(" -v " + p.Version)
	}
	installCmd.WriteString(" && rm -f install.sh")

	// Execute the command to install Chef Client
	return p.runCommand(o, comm, installCmd.String())
}

func (p *Provisioner) sshCreateConfigFiles(
	o terraform.UIOutput,
	comm communicator.Communicator) error {
	// Make sure the config directory exists
	cmd := fmt.Sprintf("mkdir -p %q", linuxConfDir)
	if err := p.runCommand(o, comm, cmd); err != nil {
		return err
	}

	// Make sure we have enough rights to upload the files if using sudo
	if !p.PreventSudo {
		if err := p.runCommand(o, comm, "chmod 777 "+linuxConfDir); err != nil {
			return err
		}
	}

	if err := p.deployConfigFiles(o, comm, linuxConfDir); err != nil {
		return err
	}

	// When done copying the files restore the rights and make sure root is owner
	if !p.PreventSudo {
		if err := p.runCommand(o, comm, "chmod 755 "+linuxConfDir); err != nil {
			return err
		}
		if err := p.runCommand(o, comm, "chown -R root.root "+linuxConfDir); err != nil {
			return err
		}
	}

	return nil
}
