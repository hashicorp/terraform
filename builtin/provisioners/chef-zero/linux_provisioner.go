package chef

import (
	"fmt"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
	"path"
	"strings"
)

const (
	chmod      = "find %s -maxdepth 1 -type f -exec /bin/chmod %d {} +"
	installURL = "https://omnitruck.chef.io/install.sh"
)

func (p *provisioner) linuxInstallChefClient(o terraform.UIOutput, comm communicator.Communicator) error {
	// Build up the command prefix
	prefix := ""
	if p.HTTPProxy != "" {
		prefix += fmt.Sprintf("http_proxy='%s' ", p.HTTPProxy)
	}
	if p.HTTPSProxy != "" {
		prefix += fmt.Sprintf("https_proxy='%s' ", p.HTTPSProxy)
	}
	if len(p.NOProxy) > 0 {
		prefix += fmt.Sprintf("no_proxy='%s' ", strings.Join(p.NOProxy, ","))
	}

	// First download the install.sh script from Chef
	err := p.runCommand(o, comm, fmt.Sprintf("%scurl -LO %s", prefix, installURL))
	if err != nil {
		return err
	}

	// Then execute the install.sh scrip to download and install Chef Client
	err = p.runCommand(o, comm, fmt.Sprintf("%sbash ./install.sh -v %q -c %s", prefix, p.Version, p.Channel))
	if err != nil {
		return err
	}

	// And finally cleanup the install.sh script again
	return p.runCommand(o, comm, fmt.Sprintf("%srm -f install.sh", prefix))
}

func (p *provisioner) preUploadDirectory(o terraform.UIOutput, comm communicator.Communicator, dir string) error {

	// Make sure the config directory exists
	if err := p.runCommand(o, comm, "mkdir -p "+dir); err != nil {
		return err
	}

	// Make sure we have enough rights to upload the files if using sudo
	if p.useSudo {
		if err := p.runCommand(o, comm, "chmod 777 "+dir); err != nil {
			return err
		}
		if err := p.runCommand(o, comm, fmt.Sprintf(chmod, dir, 666)); err != nil {
			return err
		}
	}
	return nil
}

func (p *provisioner) postUploadDirectory(o terraform.UIOutput, comm communicator.Communicator, dir string) error {
	// When done copying the hints restore the rights and make sure root is owner
	if p.useSudo {
		if err := p.runCommand(o, comm, "chmod 755 "+dir); err != nil {
			return err
		}
		if err := p.runCommand(o, comm, fmt.Sprintf(chmod, dir, 600)); err != nil {
			return err
		}
		if err := p.runCommand(o, comm, "chown -R root.root "+dir); err != nil {
			return err
		}
	}
	return nil
}

func (p *provisioner) linuxCreateConfigFiles(o terraform.UIOutput, comm communicator.Communicator) error {

	// Make sure we have enough rights to upload the files if using sudo
	if err := p.preUploadDirectory(o, comm, linuxConfDir); err != nil {
		return err
	}

	// Make sure the hits directory exists
	configDirs := []string{"data_bags", "nodes", "roles", "dna", "environments", "cookbooks"}

	if len(p.OhaiHints) > 0 {
		configDirs = append(configDirs, "ohai/hints")
	}

	if err := p.prepareConfigFiles(o, comm, linuxConfDir); err != nil {
		return err
	}

	for _, dir := range configDirs {
		configDir := path.Join(linuxConfDir, dir)

		o.Output("Preparing to upload " + configDir)
		if err := p.preUploadDirectory(o, comm, configDir); err != nil {
			return err
		}

		if dir == "ohai/hints" {
			if err := p.deployOhaiHints(o, comm, configDir); err != nil {
				return err
			}
		}

		o.Output("Deploying " + configDir)
		if err := p.deployDirectoryFiles(o, comm, linuxConfDir, dir); err != nil {
			return err
		}
	}

	for _, dir := range configDirs {
		configDir := path.Join(linuxConfDir, dir)

		if err := p.postUploadDirectory(o, comm, configDir); err != nil {
			return err
		}
	}

	if err := p.postUploadDirectory(o, comm, linuxConfDir); err != nil {
		return err
	}

	return nil
}
