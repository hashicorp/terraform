package habitat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/terraform/communicator/remote"
)

const installURL = "https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"

var systemdUnit = template.Must(template.New("hab-supervisor.service").Parse(`[Unit]
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

[Install]
WantedBy=default.target`))

func (p *provisioner) linuxInstallHabitat() error {
	var err error

	// Download the hab installer
	if err := p.linuxRunCommand(fmt.Sprintf("curl --silent -L0 %s > install.sh", installURL)); err != nil {
		return err
	}

	// Run the install script
	command := "bash ./install.sh"
	if p.Version != "" {
		command = fmt.Sprintf("bash ./install.sh -v %s", p.Version)
	}

	if err = p.linuxRunCommand(command); err != nil {
		return err
	}

	// Accept the license
	if p.AcceptLicense {
		if err := p.linuxRunCommand("HAB_LICENSE=accept hab -V"); err != nil {
			return err
		}
	}

	// Create the hab user
	if err = p.createHabUser(); err != nil {
		return err
	}

	// Cleanup the installer
	return p.linuxRunCommand("rm -f install.sh")
}

func (p *provisioner) createHabUser() error {
	// Install busybox to get us the user tools we need
	if err := p.linuxRunCommand("hab install core/busybox"); err != nil {
		return err
	}

	// Check for existing hab user
	if err := p.linuxRunCommand("hab pkg exec core/busybox id hab"); err != nil {
		p.ui.Output("No existing hab user detected, creating...")
		return p.linuxRunCommand(`hab pkg exec core/busybox adduser -D -g "" hab`)
	}

	return nil
}

func (p *provisioner) linuxStartHabitat() error {
	// Install the supervisor first
	var command string
	if p.Version == "" {
		command = "hab install core/hab-sup"
	} else {
		command = fmt.Sprintf("hab install core/hab-sup/%s", p.Version)
	}

	if err := p.linuxRunCommand(command); err != nil {
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

	if p.HttpDisable {
		options += " --http-disable"
	}

	if p.AutoUpdate {
		options += " --auto-update"
	}

	p.SupOptions = options

	// Start hab depending on service type
	switch p.ServiceType {
	case "unmanaged":
		return p.linuxStartHabitatUnmanaged(options)
	case "systemd":
		return p.linuxStartHabitatSystemd(options)
	default:
		return errors.New("unsupported service type")
	}
}

// This func is a little different than the others since we need to expose HAB_AUTH_TOKEN to a shell
// sub-process that's actually running the supervisor.
func (p *provisioner) linuxStartHabitatUnmanaged(options string) error {
	var token string

	// Create the sup directory for the log file
	if err := p.linuxRunCommand("mkdir -p /hab/sup/default && chmod o+w /hab/sup/default"); err != nil {
		return err
	}

	// Set HAB_AUTH_TOKEN if provided
	if p.BuilderAuthToken != "" {
		token = fmt.Sprintf("env HAB_AUTH_TOKEN=%s ", p.BuilderAuthToken)
	}

	return p.linuxRunCommand(fmt.Sprintf("(%ssetsid hab sup run%s > /hab/sup/default/sup.log 2>&1 <&1 &) ; sleep 1", token, options))
}

func (p *provisioner) linuxStartHabitatSystemd(options string) error {
	var buf bytes.Buffer
	err := systemdUnit.Execute(&buf, p)
	if err != nil {
		return fmt.Errorf("error executing %s.service template: %s", p.ServiceName, err)
	}

	if err := p.linuxUploadSystemdUnit(&buf); err != nil {
		return err
	}

	return p.linuxRunCommand(fmt.Sprintf("systemctl enable %s && systemctl start %s", p.ServiceName, p.ServiceName))
}

func (p *provisioner) linuxUploadSystemdUnit(contents *bytes.Buffer) error {
	destination := fmt.Sprintf("/etc/systemd/system/%s.service", p.ServiceName)

	if p.UseSudo {
		tempPath := fmt.Sprintf("/tmp/%s.service", p.ServiceName)
		if err := p.comm.Upload(tempPath, contents); err != nil {
			return err
		}

		return p.linuxRunCommand(fmt.Sprintf("mv %s %s", tempPath, destination))
	}

	return p.comm.Upload(destination, contents)
}

func (p *provisioner) linuxUploadRingKey() error {
	return p.linuxRunCommand(fmt.Sprintf(`echo -e "%s" | hab ring key import`, p.RingKeyContent))
}

func (p *provisioner) linuxUploadCtlSecret() error {
	destination := "/hab/sup/default/CTL_SECRET"

	// Create the destination directory
	err := p.linuxRunCommand(fmt.Sprintf("mkdir -p %s", filepath.Dir(destination)))
	if err != nil {
		return err
	}

	keyContent := strings.NewReader(p.CtlSecret)
	if p.UseSudo {
		tempPath := "/tmp/CTL_SECRET"
		if err := p.comm.Upload(tempPath, keyContent); err != nil {
			return err
		}

		return p.linuxRunCommand(fmt.Sprintf("chown root:root %s && chmod 0600 %s && mv %s %s", tempPath, tempPath, tempPath, destination))
	}

	return p.comm.Upload(destination, keyContent)
}

//
// Habitat Services
//

func (p *provisioner) linuxStartOrReconfigureHabitatService(service Service) error {
	info, err := p.linuxGetHabitatServiceInfo(service)

	if err != nil {
		// If we're unable to get the service information we either haven't
		// started the service, the HTTP API is disabled, the API payload
		// is invalid, or there was an issue running curl, eg: it's not
		// installed. In any of these cases we'll attempt to start the
		// service.
		p.ui.Output(fmt.Sprintf("Unable to determine state of %s: %v", service.Name, err))
		return p.linuxStartHabitatService(service)
	}

	return p.linuxReconfigureHabitatService(info, service)
}

func (p *provisioner) linuxStartHabitatService(service Service) error {
	p.ui.Output(fmt.Sprintf("Starting %s", service.Name))

	var options string

	if err := p.linuxInstallHabitatPackage(service); err != nil {
		return err
	}
	if err := p.uploadUserTOML(service); err != nil {
		return err
	}

	// Upload service group key
	if service.ServiceGroupKey != "" {
		err := p.uploadServiceGroupKey(service.ServiceGroupKey)
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

	return p.linuxRunCommand(fmt.Sprintf("hab svc load %s %s", service.Name, options))
}

func (p *provisioner) linuxReconfigureHabitatService(info *ServiceInfo, service Service) error {
	upToDate, err := info.Equal(service)
	if err != nil {
		return err
	}

	if upToDate {
		p.ui.Output(fmt.Sprintf("%s is up-to-date", service.Name))
		// Make sure we always have up-to-date TOML and service group encryption
		if err := p.uploadUserTOML(service); err != nil {
			return err
		}

		// Upload service group key
		if service.ServiceGroupKey != "" {
			err := p.uploadServiceGroupKey(service.ServiceGroupKey)
			if err != nil {
				return err
			}
		}

		return nil
	}

	p.ui.Output(fmt.Sprintf("Reloading %s because it has diverged", service.Name))
	err = p.linuxRunCommand(fmt.Sprintf("hab svc unload %s", service.Name))
	if err != nil {
		return err
	}

	return p.linuxStartHabitatService(service)
}

func (p *provisioner) linuxGetHabitatServiceInfo(service Service) (*ServiceInfo, error) {
	var err error

	if p.HttpDisable {
		p.ui.Output("Unable to determine Habitat service state because the Habitat supervisor HTTP API is disabled")
		return nil, errors.New("Habitat supervisor metadata API is disabled")
	}

	supHTTPAddr := p.ListenHTTP
	if supHTTPAddr == "" {
		supHTTPAddr = "127.0.0.1:9631"
	}

	group := "default"
	if service.Group != "" {
		group = service.Group
	}

	name := service.Name
	parts := strings.Split(service.Name, "/")
	if len(parts) == 2 {
		name = parts[1]
	}

	p.ui.Output(fmt.Sprintf("Getting %s state from the Habitat supervisor HTTP API", service.Name))
	cmd := fmt.Sprintf("curl -s http://%s/services/%s/%s", supHTTPAddr, name, group)

	// Sometimes it can take a few seconds for the habitat supervisor API to
	// be available, in which case we'll retry a few times with a bit of backoff.
	var stdout string
	for i := 1; i < 4; i++ {
		stdout, _, err = p.linuxOutputCommand(cmd)

		if err == nil {
			break
		}

		exitErr := &remote.ExitError{}
		if errors.As(err, &exitErr) {
			if exitErr.ExitStatus == 7 { // CURLE_COULDNT_CONNECT
				time.Sleep(time.Duration(i) * time.Second)
				continue
			}
		}

		break
	}
	if err != nil {
		return nil, fmt.Errorf("unable to access the Habitat supervisor metadata API: %w", err)
	}

	info := &ServiceInfo{}
	err = json.Unmarshal([]byte(stdout), info)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Habitat supervisor metadata API response: %w", err)
	}

	return info, err
}

// In the future we'll remove the dedicated install once the synchronous load feature in hab-sup is
// available. Until then we install here to provide output and a noisy failure mechanism because
// if you install with the pkg load, it occurs asynchronously and fails quietly.
func (p *provisioner) linuxInstallHabitatPackage(service Service) error {
	var options string

	if service.Channel != "" {
		options += fmt.Sprintf(" --channel %s", service.Channel)
	}

	if service.URL != "" {
		options += fmt.Sprintf(" --url %s", service.URL)
	}

	return p.linuxRunCommand(fmt.Sprintf("hab pkg install %s %s", service.Name, options))
}

func (p *provisioner) uploadServiceGroupKey(key string) error {
	keyName := strings.Split(key, "\n")[1]
	p.ui.Output("Uploading service group key: " + keyName)
	keyFileName := fmt.Sprintf("%s.box.key", keyName)
	destPath := path.Join("/hab/cache/keys", keyFileName)
	keyContent := strings.NewReader(key)
	if p.UseSudo {
		tempPath := path.Join("/tmp", keyFileName)
		if err := p.comm.Upload(tempPath, keyContent); err != nil {
			return err
		}

		return p.linuxRunCommand(fmt.Sprintf("mv %s %s", tempPath, destPath))
	}

	return p.comm.Upload(destPath, keyContent)
}

func (p *provisioner) uploadUserTOML(service Service) error {
	// Create the hab svc directory to lay down the user.toml before loading the service
	p.ui.Output(fmt.Sprintf("Uploading user.toml for %s", service.Name))
	destDir := fmt.Sprintf("/hab/user/%s/config", service.getPackageName(service.Name))
	if err := p.linuxRunCommand(fmt.Sprintf("mkdir -p %s", destDir)); err != nil {
		return err
	}

	userToml := strings.NewReader(service.UserTOML)

	if p.UseSudo {
		checksum := service.getServiceNameChecksum()
		if err := p.comm.Upload(fmt.Sprintf("/tmp/user-%s.toml", checksum), userToml); err != nil {
			return err
		}
		command := fmt.Sprintf("chmod o-r /tmp/user-%s.toml && mv /tmp/user-%s.toml %s/user.toml", checksum, checksum, destDir)
		return p.linuxRunCommand(command)
	}

	return p.comm.Upload(path.Join(destDir, "user.toml"), userToml)
}

func (p *provisioner) linuxRunCommand(command string) error {
	return p.runCommand(p.linuxExpandCommand(command))
}

func (p *provisioner) linuxOutputCommand(command string) (string, string, error) {
	return p.outputCommand(p.linuxExpandCommand(command))
}

func (p *provisioner) linuxExpandCommand(command string) string {
	// Always set HAB_NONINTERACTIVE & HAB_NOCOLORING
	env := "env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true"

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
