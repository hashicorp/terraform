package habitat

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const installURL = "https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"
const systemdUnit = `[Unit]
Description=Habitat Supervisor

[Service]
ExecStart=/bin/hab sup run{{ .SupOptions }}
Restart=on-failure
{{ if .GatewayAuthToken -}}
Environment="HAB_SUP_GATEWAY_AUTH_TOKEN={{ .GatewayAuthToken }}"
{{ end -}}
{{ if .BuilderAuthToken -}}
Environment="HAB_AUTH_TOKEN={{ .BuilderAuthToken }}"
{{ end -}}
{{ if .License -}}
Environment="HAB_LICENSE={{ .License }}"
{{ end -}}

[Install]
WantedBy=default.target
`

func (p *provisioner) linuxInstallHabitat(o terraform.UIOutput, comm communicator.Communicator) error {
	// Download the hab installer
	if err := p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("curl --silent -L0 %s > install.sh", installURL))); err != nil {
		return err
	}

	// Run the hab install script
	var command string
	if p.Version == "" {
		command = p.linuxGetCommand(fmt.Sprintf("bash ./install.sh "))
	} else {
		command = p.linuxGetCommand(fmt.Sprintf("bash ./install.sh -v %s", p.Version))
	}

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	// Create the hab user
	if err := p.createHabUser(o, comm); err != nil {
		return err
	}

	// Cleanup the installer
	return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("rm -f install.sh")))
}

func (p *provisioner) createHabUser(o terraform.UIOutput, comm communicator.Communicator) error {
	var addUser bool

	// Install busybox to get us the user tools we need
	if err := p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("hab install core/busybox"))); err != nil {
		return err
	}

	// Check for existing hab user
	if err := p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("hab pkg exec core/busybox id hab"))); err != nil {
		o.Output("No existing hab user detected, creating...")
		addUser = true
	}

	if addUser {
		return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("hab pkg exec core/busybox adduser -D -g \"\" hab")))
	}

	return nil
}

func (p *provisioner) linuxStartHabitat(o terraform.UIOutput, comm communicator.Communicator) error {
	// Install the supervisor first
	var command string
	if p.Version == "" {
		command += p.linuxGetCommand(fmt.Sprintf("hab install core/hab-sup"))
	} else {
		command += p.linuxGetCommand(fmt.Sprintf("hab install core/hab-sup/%s", p.Version))
	}

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	// Build up supervisor options
	options := ""
	if p.PermanentPeer {
		options += " --permanent-peer"
	}

	if p.ListenCtl != "" {
		options += fmt.Sprintf(" --listen-ctl %s", p.ListenCtl)
	}

	if p.ListenGossip != "" {
		options += fmt.Sprintf(" --listen-gossip %s", p.ListenGossip)
	}

	if p.ListenHTTP != "" {
		options += fmt.Sprintf(" --listen-http %s", p.ListenHTTP)
	}

	if p.Peer != "" {
		options += fmt.Sprintf(" %s", p.Peer)
	}

	if len(p.Peers) > 0 {
		if len(p.Peers) == 1 {
			options += fmt.Sprintf(" --peer %s", p.Peers[0])
		} else {
			options += fmt.Sprintf(" --peer %s", strings.Join(p.Peers, " --peer "))
		}
	}

	if p.RingKey != "" {
		options += fmt.Sprintf(" --ring %s", p.RingKey)
	}

	if p.URL != "" {
		options += fmt.Sprintf(" --url %s", p.URL)
	}

	if p.Channel != "" {
		options += fmt.Sprintf(" --channel %s", p.Channel)
	}

	if p.Events != "" {
		options += fmt.Sprintf(" --events %s", p.Events)
	}

	if p.Organization != "" {
		options += fmt.Sprintf(" --org %s", p.Organization)
	}

	if p.HttpDisable == true {
		options += fmt.Sprintf(" --http-disable")
	}

	if p.AutoUpdate == true {
		options += fmt.Sprintf(" --auto-update")
	}

	p.SupOptions = options

	// Start hab depending on service type
	switch p.ServiceType {
	case "unmanaged":
		return p.linuxStartHabitatUnmanaged(o, comm, options)
	case "systemd":
		return p.linuxStartHabitatSystemd(o, comm, options)
	default:
		return errors.New("unsupported service type")
	}
}

// This func is a little different than the others since we need to expose HAB_AUTH_TOKEN and HAB_LICENSE to a shell
// sub-process that's actually running the supervisor.
func (p *provisioner) linuxStartHabitatUnmanaged(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	var token string
	var license string

	// Create the sup directory for the log file
	if err := p.runCommand(o, comm, p.linuxGetCommand("mkdir -p /hab/sup/default && chmod o+w /hab/sup/default")); err != nil {
		return err
	}

	// Set HAB_AUTH_TOKEN if provided
	if p.BuilderAuthToken != "" {
		token = fmt.Sprintf("HAB_AUTH_TOKEN=%s ", p.BuilderAuthToken)
	}

	// Set HAB_LICENSE if provided
	if p.License != "" {
		license = fmt.Sprintf("HAB_LICENSE=%s ", p.License)
	}

	return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("(env %s%s setsid hab sup run%s > /hab/sup/default/sup.log 2>&1 <&1 &) ; sleep 1", token, license, options)))
}

func (p *provisioner) linuxStartHabitatSystemd(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	// Create a new template and parse the client config into it
	unitString := template.Must(template.New("hab-supervisor.service").Parse(systemdUnit))

	var buf bytes.Buffer
	err := unitString.Execute(&buf, p)
	if err != nil {
		return fmt.Errorf("error executing %s.service template: %s", p.ServiceName, err)
	}

	if err := p.linuxUploadSystemdUnit(o, comm, &buf); err != nil {
		return err
	}

	return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("systemctl enable %s && systemctl start %s", p.ServiceName, p.ServiceName)))
}

func (p *provisioner) linuxUploadSystemdUnit(o terraform.UIOutput, comm communicator.Communicator, contents *bytes.Buffer) error {
	destination := fmt.Sprintf("/etc/systemd/system/%s.service", p.ServiceName)

	if p.UseSudo {
		tempPath := fmt.Sprintf("/tmp/%s.service", p.ServiceName)
		if err := comm.Upload(tempPath, contents); err != nil {
			return err
		}

		return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("mv %s %s", tempPath, destination)))
	}

	return comm.Upload(destination, contents)
}

func (p *provisioner) linuxUploadRingKey(o terraform.UIOutput, comm communicator.Communicator) error {
	return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf(`echo -e "%s" | hab ring key import`, p.RingKeyContent)))
}

func (p *provisioner) linuxUploadCtlSecret(o terraform.UIOutput, comm communicator.Communicator) error {
	destination := fmt.Sprintf("/hab/sup/default/CTL_SECRET")
	// Create the destination directory
	err := p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("mkdir -p %s", filepath.Dir(destination))))
	if err != nil {
		return err
	}

	keyContent := strings.NewReader(p.CtlSecret)
	if p.UseSudo {
		tempPath := fmt.Sprintf("/tmp/CTL_SECRET")
		if err := comm.Upload(tempPath, keyContent); err != nil {
			return err
		}

		return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("mv %s %s && chown root:root %s && chmod 0600 %s", tempPath, destination, destination, destination)))
	}

	return comm.Upload(destination, keyContent)
}

//
// Habitat Services
//
func (p *provisioner) linuxStartHabitatService(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	var options string

	if err := p.linuxInstallHabitatPackage(o, comm, service); err != nil {
		return err
	}
	if err := p.uploadUserTOML(o, comm, service); err != nil {
		return err
	}

	// Upload service group key
	if service.ServiceGroupKey != "" {
		err := p.uploadServiceGroupKey(o, comm, service.ServiceGroupKey)
		if err != nil {
			return err
		}
	}

	if service.Topology != "" {
		options += fmt.Sprintf(" --topology %s", service.Topology)
	}

	if service.Strategy != "" {
		options += fmt.Sprintf(" --strategy %s", service.Strategy)
	}

	if service.Channel != "" {
		options += fmt.Sprintf(" --channel %s", service.Channel)
	}

	if service.URL != "" {
		options += fmt.Sprintf(" --url %s", service.URL)
	}

	if service.Group != "" {
		options += fmt.Sprintf(" --group %s", service.Group)
	}

	for _, bind := range service.Binds {
		options += fmt.Sprintf(" --bind %s", bind.toBindString())
	}

	return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("hab svc load %s %s", service.Name, options)))
}

// In the future we'll remove the dedicated install once the synchronous load feature in hab-sup is
// available. Until then we install here to provide output and a noisy failure mechanism because
// if you install with the pkg load, it occurs asynchronously and fails quietly.
func (p *provisioner) linuxInstallHabitatPackage(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	var options string

	if service.Channel != "" {
		options += fmt.Sprintf(" --channel %s", service.Channel)
	}

	if service.URL != "" {
		options += fmt.Sprintf(" --url %s", service.URL)
	}

	return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("hab pkg install %s %s", service.Name, options)))
}

func (p *provisioner) uploadServiceGroupKey(o terraform.UIOutput, comm communicator.Communicator, key string) error {
	keyName := strings.Split(key, "\n")[1]
	o.Output("Uploading service group key: " + keyName)
	keyFileName := fmt.Sprintf("%s.box.key", keyName)
	destPath := path.Join("/hab/cache/keys", keyFileName)
	keyContent := strings.NewReader(key)
	if p.UseSudo {
		tempPath := path.Join("/tmp", keyFileName)
		if err := comm.Upload(tempPath, keyContent); err != nil {
			return err
		}

		return p.runCommand(o, comm, p.linuxGetCommand(fmt.Sprintf("mv %s %s", tempPath, destPath)))
	}

	return comm.Upload(destPath, keyContent)
}

func (p *provisioner) uploadUserTOML(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	// Create the hab svc directory to lay down the user.toml before loading the service
	o.Output("Uploading user.toml for service: " + service.Name)
	destDir := fmt.Sprintf("/hab/user/%s/config", service.getPackageName(service.Name))
	command := p.linuxGetCommand(fmt.Sprintf("mkdir -p %s", destDir))
	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	userToml := strings.NewReader(service.UserTOML)

	if p.UseSudo {
		if err := comm.Upload(fmt.Sprintf("/tmp/user-%s.toml", service.getServiceNameChecksum()), userToml); err != nil {
			return err
		}
		command = p.linuxGetCommand(fmt.Sprintf("mv /tmp/user-%s.toml %s/user.toml", service.getServiceNameChecksum(), destDir))
		return p.runCommand(o, comm, command)
	}

	return comm.Upload(path.Join(destDir, "user.toml"), userToml)
}

func (p *provisioner) linuxGetCommand(command string) string {
	// Always set HAB_NONINTERACTIVE & HAB_NOCOLORING
	env := fmt.Sprintf("env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true")

	// Set license acceptance
	if p.License != "" {
		env += fmt.Sprintf(" HAB_LICENSE=%s", p.License)
	}

	// Set builder auth token
	if p.BuilderAuthToken != "" {
		env += fmt.Sprintf(" HAB_AUTH_TOKEN=%s", p.BuilderAuthToken)
	}

	if p.UseSudo {
		command = fmt.Sprintf("%s sudo -E /bin/bash -c '%s'", env, command)
	} else {
		command = fmt.Sprintf("%s /bin/bash -c '%s'", env, command)
	}

	return command
}
