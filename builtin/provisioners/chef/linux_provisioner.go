package chef

import (
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	installURL = "https://www.chef.io/chef/install.sh"
)

func (p *Provisioner) linuxInstallChefClient(
	o terraform.UIOutput,
	comm communicator.Communicator) error {

	// Build up the command prefix
	prefix := ""
	if p.HTTPProxy != "" {
		prefix += fmt.Sprintf("http_proxy='%s' ", p.HTTPProxy)
	}
	if p.HTTPSProxy != "" {
		prefix += fmt.Sprintf("https_proxy='%s' ", p.HTTPSProxy)
	}
	if p.NOProxy != nil {
		prefix += fmt.Sprintf("no_proxy='%s' ", strings.Join(p.NOProxy, ","))
	}

	// First download the install.sh script from Chef
	err := p.runCommand(o, comm, fmt.Sprintf("%scurl -LO %s", prefix, installURL))
	if err != nil {
		return err
	}

	// Then execute the install.sh scrip to download and install Chef Client
	err = p.runCommand(o, comm, fmt.Sprintf("%sbash ./install.sh -v %q", prefix, p.Version))
	if err != nil {
		return err
	}

	// And finally cleanup the install.sh script again
	return p.runCommand(o, comm, fmt.Sprintf("%srm -f install.sh", prefix))
}

func (p *Provisioner) linuxCreateConfigFiles(
	o terraform.UIOutput,
	comm communicator.Communicator) error {
	// Make sure the config directory exists
	if err := p.runCommand(o, comm, "mkdir -p "+linuxConfDir); err != nil {
		return err
	}

	// Make sure we have enough rights to upload the files if using sudo
	if p.useSudo {
		if err := p.runCommand(o, comm, "chmod 777 "+linuxConfDir); err != nil {
			return err
		}
	}

	if err := p.deployConfigFiles(o, comm, linuxConfDir); err != nil {
		return err
	}

	if len(p.OhaiHints) > 0 {
		// Make sure the hits directory exists
		hintsDir := path.Join(linuxConfDir, "ohai/hints")
		if err := p.runCommand(o, comm, "mkdir -p "+hintsDir); err != nil {
			return err
		}

		// Make sure we have enough rights to upload the hints if using sudo
		if p.useSudo {
			if err := p.runCommand(o, comm, "chmod 777 "+hintsDir); err != nil {
				return err
			}
		}

		if err := p.deployOhaiHints(o, comm, hintsDir); err != nil {
			return err
		}

		// When done copying the hints restore the rights and make sure root is owner
		if p.useSudo {
			if err := p.runCommand(o, comm, "chmod 755 "+hintsDir); err != nil {
				return err
			}
			if err := p.runCommand(o, comm, "chown -R root.root "+hintsDir); err != nil {
				return err
			}
		}
	}

	// When done copying all files restore the rights and make sure root is owner
	if p.useSudo {
		if err := p.runCommand(o, comm, "chmod 755 "+linuxConfDir); err != nil {
			return err
		}
		if err := p.runCommand(o, comm, "chown -R root.root "+linuxConfDir); err != nil {
			return err
		}
	}

	return nil
}
