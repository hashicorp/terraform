package puppet

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform/communicator/remote"
)

const (
	getHostByName = "([System.Net.Dns]::GetHostByName(($env:computerName))).Hostname"
	domainQuery   = "(Get-WmiObject -Query 'select DNSDomain from Win32_NetworkAdapterConfiguration where IPEnabled = True').DNSDomain"
)

func (p *provisioner) windowsUploadFile(f io.Reader, dir string, filename string) error {
	_, err := p.runCommand("powershell.exe new-item -itemtype directory -force -path " + dir)
	if err != nil {
		return fmt.Errorf("Failed to make directory %s: %s", dir, err)
	}

	return p.comm.Upload(dir+"\\"+filename, f)
}

func (p *provisioner) windowsDefaultCertname() (string, error) {
	certname, err := p.runCommand(fmt.Sprintf(`powershell -Command "& {%s}"`, getHostByName))
	if err != nil {
		return "", err
	}

	// Sometimes System.Net.Dns::GetHostByName does not return a full FQDN, so
	// we have to look up the domain separately.
	if strings.Contains(certname, ".") {
		return certname, nil
	}

	domain, err := p.runCommand(fmt.Sprintf(`powershell -Command "& {%s}"`, domainQuery))
	if err != nil {
		return "", err
	}

	return strings.ToLower(certname + "." + domain), nil
}

func (p *provisioner) windowsInstallPuppetAgent() error {
	_, err := p.runCommand(fmt.Sprintf(
		`powershell -Command "& {[Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}; `+
			`(New-Object System.Net.WebClient).DownloadFile('https://%s:8140/packages/current/install.ps1', `+
			`'install.ps1')}"`,
		p.Server,
	))
	if err != nil {
		return err
	}

	_, err = p.runCommand(`powershell -Command "& .\install.ps1 -PuppetServiceEnsure stopped"`)

	return err
}

func (p *provisioner) windowsRunPuppetAgent() error {
	_, err := p.runCommand(fmt.Sprintf("puppet agent --test --server %s --environment %s", p.Server, p.Environment))
	if err != nil {
		errStruct, _ := err.(*remote.ExitError)
		if errStruct.ExitStatus == 2 {
			return nil
		}
	}

	return err
}
