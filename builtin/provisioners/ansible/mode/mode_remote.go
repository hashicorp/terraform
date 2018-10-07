package mode

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	linereader "github.com/mitchellh/go-linereader"
	"github.com/radekg/terraform-provisioner-ansible/types"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	uuid "github.com/satori/go.uuid"
)

const installerProgramTemplate = `#!/usr/bin/env sh
if [ -z "$(which ansible-playbook)" ]; then
  
  # only check the cloud boot finished if the directory exists
  if [ -d /var/lib/cloud/instance ]; then
    until [[ -f /var/lib/cloud/instance/boot-finished ]]; do
      sleep 1
    done
  fi

  # install dependencies
  if [[ -f /etc/redhat-release ]]; then
    yum update -y \
    && yum groupinstall -y "Development Tools" \
    && yum install -y python-devel
  else
    apt-get update \
    && apt-get install -y build-essential python-dev
  fi

  # install pip, if necessary
  if [ -z "$(which pip)" ]; then
    curl https://bootstrap.pypa.io/get-pip.py | sudo python
  fi

  # install ansible
  pip install {{ .AnsibleVersion}}

else

  expected_version="{{ .AnsibleVersion}}"
  installed_version=$(ansible-playbook --version | head -n1 | awk '{print $2}')
  installed_version="ansible==$installed_version"
  if [[ "$expected_version" = *"=="* ]]; then
    if [ "$expected_version" != "$installed_version" ]; then
      pip install $expected_version
    fi
  fi
  
fi
`

type inventoryTemplateRemoteData struct {
	Hosts  []string
	Groups []string
}

const inventoryTemplateRemote = `{{$top := . -}}
{{range .Hosts -}}
{{.}} ansible_connection=local
{{end}}

{{range .Groups -}}
[{{.}}]
{{range $top.Hosts -}}
{{.}} ansible_connection=local
{{end}}

{{end}}`

// RemoteMode represents remote provisioner mode.
type RemoteMode struct {
	o              terraform.UIOutput
	comm           communicator.Communicator
	connInfo       *connectionInfo
	remoteSettings *types.RemoteSettings
}

type ansibleInstaller struct {
	AnsibleVersion string
}

// NewRemoteMode returns configured remote mode provisioner.
func NewRemoteMode(o terraform.UIOutput, s *terraform.InstanceState, remoteSettings *types.RemoteSettings) (*RemoteMode, error) {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return nil, err
	}

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

	return &RemoteMode{
		o:              o,
		comm:           comm,
		connInfo:       connInfo,
		remoteSettings: remoteSettings,
	}, nil
}

// Run executes remote provisioning process.
func (v *RemoteMode) Run(plays []*types.Play) error {
	// Wait and retry until we establish the connection
	err := v.retryFunc(v.comm.Timeout(), func() error {
		return v.comm.Connect(v.o)
	})
	if err != nil {
		return err
	}
	defer v.comm.Disconnect()

	err = v.deployAnsibleData(plays)

	if err != nil {
		v.o.Output(fmt.Sprintf("%+v", err))
		return err
	}

	if !v.remoteSettings.SkipInstall() {
		if err := v.installAnsible(v.remoteSettings); err != nil {
			return err
		}
	}

	for _, play := range plays {
		command, err := play.ToCommand(types.LocalModeAnsibleArgs{Username: v.connInfo.User})
		if err != nil {
			return err
		}
		v.o.Output(fmt.Sprintf("running command: %s", command))
		if err := v.runCommandSudo(command); err != nil {
			return err
		}
	}

	if !v.remoteSettings.SkipCleanup() {
		v.cleanupAfterBootstrap()
	}

	return nil

}

// retryFunc is used to retry a function for a given duration
func (v *RemoteMode) retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("Retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}

func (v *RemoteMode) getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (v *RemoteMode) deployAnsibleData(plays []*types.Play) error {

	for _, play := range plays {
		if !play.Enabled() {
			continue
		}

		switch entity := play.Entity().(type) {
		case *types.Playbook:

			playbookPath, err := types.ResolvePath(entity.FilePath())
			if err != nil {
				return err
			}

			// playbook file is at the top level of the module
			// parse the playbook path's directory and upload the entire directory
			playbookDir := filepath.Dir(playbookPath)
			playbookDirHash := v.getMD5Hash(playbookDir)

			remotePlaybookDir := filepath.Join(v.remoteSettings.BootstrapDirectory(), playbookDirHash)
			remotePlaybookPath := filepath.Join(remotePlaybookDir, filepath.Base(playbookPath))

			if err := v.runCommandNoSudo(fmt.Sprintf("mkdir -p \"%s\"",
				v.remoteSettings.BootstrapDirectory())); err != nil {
				return err
			}

			dirExists, err := v.checkRemoteDirExists(remotePlaybookDir)
			if err != nil {
				return err
			}
			if dirExists {
				v.o.Output(fmt.Sprintf("The playbook '%s' directory '%s' has been already uploaded.", entity.FilePath(), playbookDir))
			} else {
				v.o.Output(fmt.Sprintf("Uploading the parent directory '%s' of playbook '%s' to '%s'...", playbookDir, entity.FilePath(), remotePlaybookDir))
				// upload ansible source and playbook to the host
				if err := v.comm.UploadDir(remotePlaybookDir, playbookDir); err != nil {
					return err
				}
			}

			entity.SetOverrideFilePath(remotePlaybookPath)

			// always create temp inventory:
			inventoryFile, err := v.writeInventory(remotePlaybookDir, play)
			if err != nil {
				return err
			}
			play.SetOverrideInventoryFile(inventoryFile)

			// always handle Vault ID or password file
			if len(play.VaultID()) > 0 {
				overrideVaultIDs := make([]string, 0)
				for _, vaultID := range play.VaultID() {
					uploadedVaultIDPath, err := v.uploadVaultPasswordOrIDFile(remotePlaybookDir, vaultID)
					if err != nil {
						return err
					}
					if uploadedVaultIDPath != "" {
						overrideVaultIDs = append(overrideVaultIDs, uploadedVaultIDPath)
					}
				}
				play.SetOverrideVaultID(overrideVaultIDs)
			} else {
				uploadedVaultPasswordFilePath, err := v.uploadVaultPasswordOrIDFile(remotePlaybookDir, play.VaultPasswordFile())
				if err != nil {
					return err
				}
				play.SetOverrideVaultPasswordPath(uploadedVaultPasswordFilePath)
			}

			// upload roles paths, if any:
			remoteRolesPath := make([]string, 0)
			for _, path := range entity.RolesPath() {
				resolvedPath, err := types.ResolvePath(path)
				if err != nil {
					return err
				}
				dirHash := v.getMD5Hash(resolvedPath)
				remoteDir := filepath.Join(v.remoteSettings.BootstrapDirectory(), dirHash)
				dirExists, err := v.checkRemoteDirExists(remoteDir)

				if err != nil {
					return err
				}
				if dirExists {
					v.o.Output(fmt.Sprintf("Roles path '%s' has been already uploaded.", resolvedPath))
				} else {
					v.o.Output(fmt.Sprintf("Uploading roles path '%s' to '%s'...", resolvedPath, remoteDir))
					// upload ansible source and playbook to the host
					if err := v.comm.UploadDir(remoteDir, resolvedPath); err != nil {
						return err
					}
				}
				remoteRolesPath = append(remoteRolesPath, remoteDir)
			}
			entity.SetOverrideRolesPath(remoteRolesPath)

		case *types.Module:

			moduleDirHash := v.getMD5Hash(entity.Module())
			remoteModuleDir := filepath.Join(v.remoteSettings.BootstrapDirectory(), moduleDirHash)

			if err := v.runCommandNoSudo(fmt.Sprintf("mkdir -p \"%s\"", remoteModuleDir)); err != nil {
				return err
			}

			// always handle Vault ID or password file
			if len(play.VaultID()) > 0 {
				overrideVaultIDs := make([]string, 0)
				for _, vaultID := range play.VaultID() {
					uploadedVaultIDPath, err := v.uploadVaultPasswordOrIDFile(remoteModuleDir, vaultID)
					if err != nil {
						return err
					}
					if uploadedVaultIDPath != "" {
						overrideVaultIDs = append(overrideVaultIDs, uploadedVaultIDPath)
					}
				}
				play.SetOverrideVaultID(overrideVaultIDs)
			} else {
				uploadedVaultPasswordFilePath, err := v.uploadVaultPasswordOrIDFile(remoteModuleDir, play.VaultPasswordFile())
				if err != nil {
					return err
				}
				play.SetOverrideVaultPasswordPath(uploadedVaultPasswordFilePath)
			}

			// always create temp inventory:
			inventoryFile, err := v.writeInventory(remoteModuleDir, play)
			if err != nil {
				return err
			}
			play.SetOverrideInventoryFile(inventoryFile)

		}
	}

	return nil
}

func (v *RemoteMode) installAnsible(remoteSettings *types.RemoteSettings) error {

	var installerScript *bufio.Reader
	if remoteSettings.LocalInstallerPath() != "" {

		cleanInstallerPath := filepath.Clean(remoteSettings.LocalInstallerPath())
		file, err := os.Open(cleanInstallerPath)
		if err != nil {
			return err
		}
		defer file.Close()

		v.o.Output(fmt.Sprintf("Installing Ansible using provided installer '%s'...", cleanInstallerPath))

		installerScript = bufio.NewReader(file)

	} else {

		embeddedInstaller := &ansibleInstaller{
			AnsibleVersion: "ansible",
		}

		if remoteSettings.InstallVersion() != "" {
			embeddedInstaller.AnsibleVersion = fmt.Sprintf("%s==%s",
				embeddedInstaller.AnsibleVersion,
				remoteSettings.InstallVersion())
		}

		v.o.Output(fmt.Sprintf("Installing Ansible '%s' using default installer...", embeddedInstaller.AnsibleVersion))

		t := template.Must(template.New("installer").Parse(installerProgramTemplate))
		var buf bytes.Buffer
		err := t.Execute(&buf, embeddedInstaller)
		if err != nil {
			return fmt.Errorf("Error executing 'installer' template: %s", err)
		}
		installerScript = bufio.NewReader(bytes.NewReader(buf.Bytes()))
	}

	if err := v.runCommandNoSudo(fmt.Sprintf("mkdir -p \"%s\"",
		filepath.Dir(remoteSettings.RemoteInstallerPath()))); err != nil {
		return err
	}

	v.o.Output(fmt.Sprintf("Uploading Ansible installer program to '%s'...", remoteSettings.RemoteInstallerPath()))
	if err := v.comm.UploadScript(remoteSettings.RemoteInstallerPath(), installerScript); err != nil {
		return err
	}

	if err := v.runCommandSudo(fmt.Sprintf("/bin/sh -c '\"%s\" && rm \"%s\"'",
		remoteSettings.RemoteInstallerPath(),
		remoteSettings.RemoteInstallerPath())); err != nil {
		return err
	}

	v.o.Output("Ansible installed.")
	return nil
}

func (v *RemoteMode) uploadVaultPasswordOrIDFile(destination string, source string) (string, error) {

	if source == "" {
		return "", nil
	}

	source, err := types.ResolvePath(source)
	if err != nil {
		return "", err
	}

	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(destination, fmt.Sprintf(".vault-file-%s", u1))

	v.o.Output(fmt.Sprintf("Uploading ansible vault password file / ID to '%s'...", targetPath))

	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := v.comm.Upload(targetPath, bufio.NewReader(file)); err != nil {
		return "", err
	}

	v.o.Output("Ansible vault password file uploaded.")

	return targetPath, nil
}

func (v *RemoteMode) writeInventory(destination string, play *types.Play) (string, error) {

	if play.InventoryFile() != "" {

		v.o.Output(fmt.Sprintf("Using provided inventory file '%s'...", play.InventoryFile()))
		source, err := types.ResolvePath(play.InventoryFile())
		if err != nil {
			return "", err
		}
		u1 := uuid.Must(uuid.NewV4())
		targetPath := filepath.Join(destination, fmt.Sprintf(".inventory-%s", u1))
		v.o.Output(fmt.Sprintf("Uploading provided inventory file '%s' to '%s'...", play.InventoryFile(), targetPath))

		file, err := os.Open(source)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if err := v.comm.Upload(targetPath, bufio.NewReader(file)); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory uploaded.")

		return targetPath, nil

	}

	templateData := inventoryTemplateRemoteData{
		Hosts:  ensureLocalhostInHosts(play.Hosts()),
		Groups: play.Groups(),
	}

	v.o.Output("Generating temporary ansible inventory...")
	t := template.Must(template.New("hosts").Parse(inventoryTemplateRemote))
	var buf bytes.Buffer
	err := t.Execute(&buf, templateData)
	if err != nil {
		return "", fmt.Errorf("Error executing 'hosts' template: %s", err)
	}

	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(destination, fmt.Sprintf(".inventory-%s", u1))

	v.o.Output(fmt.Sprintf("Writing temporary ansible inventory to '%s'...", targetPath))
	if err := v.comm.Upload(targetPath, bytes.NewReader(buf.Bytes())); err != nil {
		return "", err
	}

	v.o.Output("Ansible inventory written.")
	return targetPath, nil

}

func (v *RemoteMode) cleanupAfterBootstrap() {
	v.o.Output("Cleaning up after bootstrap...")
	v.runCommandNoSudo(fmt.Sprintf("rm -rf \"%s\"", v.remoteSettings.BootstrapDirectory()))
	v.o.Output("Cleanup complete.")
}

func (v *RemoteMode) checkRemoteDirExists(remoteDir string) (bool, error) {
	command := "/bin/sh -c 'if [ -d \"%s\" ]; then exit 50; fi'"
	if err := v.runCommandNoSudo(fmt.Sprintf(command, remoteDir)); err != nil {
		errDetail := strings.Split(fmt.Sprintf("%v", err), ": ")
		if errDetail[len(errDetail)-1] == "50" {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (v *RemoteMode) runCommandSudo(command string) error {
	return v.runCommand(command, true)
}

func (v *RemoteMode) runCommandNoSudo(command string) error {
	return v.runCommand(command, false)
}

func (v *RemoteMode) runCommand(command string, shouldSudo bool) error {
	// Unless prevented, prefix the command with sudo
	if shouldSudo && v.remoteSettings.UseSudo() {
		command = fmt.Sprintf("sudo %s", command)
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go v.copyOutput(outR, outDoneCh)
	go v.copyOutput(errR, errDoneCh)

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err := v.comm.Start(cmd)
	if err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*remote.ExitError); ok {
			err = fmt.Errorf(
				"Command '%q' exited with non-zero exit status: %d, reason %+v", cmd.Command, exitErr.ExitStatus, exitErr.Err)
		} else {
			err = fmt.Errorf(
				"Command '%q' failed, reason: %+v", cmd.Command, err)
		}
	}

	// Wait for output to clean up
	outW.Close()
	errW.Close()
	<-outDoneCh
	<-errDoneCh

	return err
}

func (v *RemoteMode) copyOutput(r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		v.o.Output(line)
	}
}

func ensureLocalhostInHosts(hosts []string) []string {
	found := false
	for _, host := range hosts {
		if host == "localhost" {
			found = true
			break
		}
	}
	if !found {
		newHosts := []string{"localhost"}
		for _, host := range hosts {
			newHosts = append(newHosts, host)
		}
		return newHosts
	}
	return hosts
}
