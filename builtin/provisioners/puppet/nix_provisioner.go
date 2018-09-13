package puppet

import (
	"fmt"
	"io"
	"strconv"

	"github.com/hashicorp/terraform/communicator/remote"
)

func (p *provisioner) nixUploadFile(f io.Reader, dir string, filename string) error {
	_, err := p.runCommand("mkdir -p " + dir)
	if err != nil {
		return fmt.Errorf("Failed to make directory %s: %s", dir, err)
	}

	err = p.comm.Upload("/tmp/"+filename, f)
	if err != nil {
		return fmt.Errorf("Failed to upload %s to /tmp: %s", filename, err)
	}

	_, err = p.runCommand(fmt.Sprintf("mv /tmp/%s %s/%s", filename, dir, filename))
	return err
}

func (p *provisioner) nixDefaultCertname() (string, error) {
	certname, err := p.runCommand("hostname -f")
	if err != nil {
		return "", err
	}

	return certname, nil
}

func (p *provisioner) nixInstallPuppetAgent() error {
	_, err := p.runCommand(fmt.Sprintf("curl -kO https://%s:8140/packages/current/install.bash", p.Server))
	if err != nil {
		return err
	}

	_, err = p.runCommand("bash -- ./install.bash --puppet-service-ensure stopped")
	if err != nil {
		return err
	}

	_, err = p.runCommand("rm -f install.bash")
	return err
}

func (p *provisioner) nixRunPuppetAgent() error {
	_, err := p.runCommand(fmt.Sprintf("/opt/puppetlabs/puppet/bin/puppet agent --test --server %s --environment %s", p.Server, p.Environment))

	// Puppet exits 2 if changes have been successfully made.
	if err != nil {
		errStruct, _ := err.(*remote.ExitError)
		if errStruct.ExitStatus == 2 {
			return nil
		}
	}

	return err
}

func (p *provisioner) nixIsPuppetEnterprise() (bool, error) {
	status, err := p.runCommand(fmt.Sprintf(`curl -IsLk -w "%%{http_code}" -o /dev/null https://%s:8140/packages/current/install.bash`, p.Server))
	if err != nil {
		return false, err
	}

	statusInt, err := strconv.Atoi(status)
	if err != nil {
		return false, err
	}

	return (statusInt < 400), nil
}
