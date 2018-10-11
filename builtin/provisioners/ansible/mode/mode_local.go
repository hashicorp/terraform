package mode

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/terraform/builtin/provisioners/ansible/types"
	uuid "github.com/satori/go.uuid"

	localExec "github.com/hashicorp/terraform/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/terraform"
)

// LocalMode represents local provisioner mode.
type LocalMode struct {
	o        terraform.UIOutput
	connInfo *connectionInfo
}

type inventoryTemplateLocalDataHost struct {
	Alias       string
	AnsibleHost string
}

type inventoryTemplateLocalData struct {
	Hosts  []inventoryTemplateLocalDataHost
	Groups []string
}

const inventoryTemplateLocal = `{{$top := . -}}
{{range .Hosts -}}
{{.Alias -}}
{{if ne .AnsibleHost "" -}}
{{" "}}ansible_host={{.AnsibleHost -}}
{{end -}}
{{end}}

{{range .Groups -}}
[{{.}}]
{{range $top.Hosts -}}
{{.Alias -}}
{{if ne .AnsibleHost "" -}}
{{" "}}ansible_host={{.AnsibleHost -}}
{{end -}}
{{end}}

{{end}}`

// NewLocalMode returns configured local mode provisioner.
func NewLocalMode(o terraform.UIOutput, s *terraform.InstanceState) (*LocalMode, error) {

	connType := s.Ephemeral.ConnInfo["type"]
	switch connType {
	case "ssh", "": // The default connection type is ssh, so if connType is empty use ssh
	default:
		return nil, fmt.Errorf("Currently, only SSH connection is supported")
	}

	connInfo, err := parseConnectionInfo(s)
	if err != nil {
		return nil, err
	}
	if connInfo.User == "" || connInfo.Host == "" {
		return nil, fmt.Errorf("Local mode requires a connection with username and host")
	}

	return &LocalMode{
		o:        o,
		connInfo: connInfo,
	}, nil
}

// Run executes local provisioning process.
func (v *LocalMode) Run(plays []*types.Play, ansibleSSHSettings *types.AnsibleSSHSettings) error {

	pemFile := ""
	if v.connInfo.PrivateKey != "" {
		var err error
		pemFile, err = v.writePem()
		if err != nil {
			return err
		}
		defer os.Remove(pemFile)
	}

	bastion := newBastionHostFromConnectionInfo(v.connInfo)
	target := newTargetHostFromConnectionInfo(v.connInfo)

	knownHosts := make([]string, 0)

	if bastion.inUse() {
		// wait for bastion:
		sshClient, err := bastion.connect()
		if err != nil {
			return err
		}
		defer sshClient.Close()
		if target.hostKey() == "" {
			v.o.Output(fmt.Sprintf("Host key not given, executing ssh-keyscan on bastion: %s@%s:%d",
				bastion.user(),
				bastion.host(),
				bastion.port()))
			targetKnownHosts, err := newBastionKeyScan(v.o,
				sshClient,
				target.host(),
				target.port(),
				ansibleSSHSettings.SSHKeyscanSeconds()).scan()
			if err != nil {
				return err
			}
			// ssh-keyscan gave us full lines with hosts, like this:
			// <ip> ecdsa-sha2-nistp256 AAAA...
			// <ip> ssh-rsa AAAAB...
			// <ip> ssh-ed25519 AAAAC...
			knownHosts = append(knownHosts, targetKnownHosts)
		} else {
			knownHosts = append(knownHosts, fmt.Sprintf("%s %s", target.host(), target.hostKey()))
		}
		knownHosts = append(knownHosts, fmt.Sprintf("%s %s", bastion.host(), bastion.hostKey()))
	} else {
		if target.hostKey() == "" {

			// fetchHostKey will issue an ssh Dial and update the hostKey() value
			// as with bastionKeyScan, we might ask for the host key while the instance
			// is not ready to respond to SSH, we need to retry for a number of times
			timeoutMs := ansibleSSHSettings.SSHKeyscanSeconds() * 1000
			timeSpentMs := 0
			intervalMs := 5000
			for {
				if err := target.fetchHostKey(); err != nil {
					v.o.Output(fmt.Sprintf("host key for '%s' not received yet; retrying...", target.host()))
					time.Sleep(time.Duration(intervalMs) * time.Millisecond)
					timeSpentMs = timeSpentMs + intervalMs
					if timeSpentMs > timeoutMs {
						v.o.Output(fmt.Sprintf("host key for '%s' not received within %d seconds",
							target.host(),
							ansibleSSHSettings.SSHKeyscanSeconds()))
						return err
					}
				} else {
					break
				}
			}
			if target.hostKey() == "" {
				return fmt.Errorf("expected to receive the host key for '%s', but no host key arrived", target.host())
			}
		}
		knownHosts = append(knownHosts, fmt.Sprintf("%s %s", target.host(), target.hostKey()))
	}

	knownHostsFile, err := v.writeKnownHosts(knownHosts)
	if err != nil {
		return err
	}
	defer os.Remove(knownHostsFile)

	for _, play := range plays {

		if !play.Enabled() {
			continue
		}

		inventoryFile, err := v.writeInventory(play)

		if err != nil {
			v.o.Output(fmt.Sprintf("%+v", err))
			return err
		}

		if inventoryFile != play.InventoryFile() {
			play.SetOverrideInventoryFile(inventoryFile)
			defer os.Remove(play.InventoryFile())
		}

		// we can't pass bastion instance into this function
		// we would end up with a circular import
		command, err := play.ToLocalCommand(types.LocalModeAnsibleArgs{
			Username:        v.connInfo.User,
			Port:            v.connInfo.Port,
			PemFile:         pemFile,
			KnownHostsFile:  knownHostsFile,
			BastionHost:     bastion.host(),
			BastionPemFile:  bastion.pemFile(),
			BastionPort:     bastion.port(),
			BastionUsername: bastion.user(),
		}, ansibleSSHSettings)

		if err != nil {
			return err
		}

		v.o.Output(fmt.Sprintf("running local command: %s", command))

		if err := v.runCommand(command); err != nil {
			return err
		}

	}

	return nil
}

func (v *LocalMode) writeKnownHosts(knownHosts []string) (string, error) {
	trimmedKnownHosts := make([]string, 0)
	for _, entry := range knownHosts {
		trimmedKnownHosts = append(trimmedKnownHosts, strings.TrimSpace(entry))
	}
	knownHostsFileContents := strings.Join(trimmedKnownHosts, "\n")
	file, err := ioutil.TempFile(os.TempDir(), uuid.NewV4().String())
	defer file.Close()
	if err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(file.Name(), []byte(fmt.Sprintf("%s\n", knownHostsFileContents)), 0644); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func (v *LocalMode) writePem() (string, error) {
	if v.connInfo.PrivateKey != "" {
		file, err := ioutil.TempFile(os.TempDir(), "temporary-private-key.pem")
		defer file.Close()
		if err != nil {
			return "", err
		}

		v.o.Output(fmt.Sprintf("Writing temprary PEM to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), []byte(v.connInfo.PrivateKey), 0400); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory written.")
		return file.Name(), nil
	}
	return "", nil
}

func (v *LocalMode) writeInventory(play *types.Play) (string, error) {
	if play.InventoryFile() == "" {
		if v.connInfo.Host == "" {
			return "", fmt.Errorf("Host could not be established from the connection info")
		}

		playHosts := play.Hosts()

		templateData := inventoryTemplateLocalData{
			Hosts:  make([]inventoryTemplateLocalDataHost, 0),
			Groups: play.Groups(),
		}

		if len(playHosts) > 0 {
			if playHosts[0] != "" {
				templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
					Alias:       playHosts[0],
					AnsibleHost: v.connInfo.Host,
				})
			} else {
				templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
					Alias: v.connInfo.Host,
				})
			}
		} else {
			templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
				Alias: v.connInfo.Host,
			})
		}

		v.o.Output("Generating temporary ansible inventory...")
		t := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
		var buf bytes.Buffer
		err := t.Execute(&buf, templateData)
		if err != nil {
			return "", fmt.Errorf("Error executing 'hosts' template: %s", err)
		}

		file, err := ioutil.TempFile(os.TempDir(), "temporary-ansible-inventory")
		defer file.Close()
		if err != nil {
			return "", err
		}

		v.o.Output(fmt.Sprintf("Writing temporary ansible inventory to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), buf.Bytes(), 0644); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory written.")

		return file.Name(), nil
	}

	return play.InventoryFile(), nil
}

func (v *LocalMode) runCommand(command string) error {
	localExecProvisioner := localExec.Provisioner()

	instanceState := &terraform.InstanceState{
		ID:         command,
		Attributes: make(map[string]string),
		Ephemeral: terraform.EphemeralState{
			ConnInfo: make(map[string]string),
			Type:     "local-exec",
		},
		Meta: map[string]interface{}{
			"command": command,
		},
		Tainted: false,
	}

	config := &terraform.ResourceConfig{
		ComputedKeys: make([]string, 0),
		Raw: map[string]interface{}{
			"command": command,
		},
		Config: map[string]interface{}{
			"command": command,
		},
	}

	return localExecProvisioner.Apply(v.o, instanceState, config)
}
