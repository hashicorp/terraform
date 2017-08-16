package ansible_local

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{

			"command": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ANSIBLE_FORCE_COLOR=1 PYTHONUNBUFFERED=1 ansible-playbook",
			},

			"extra_arguments": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"group_vars": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if err := validateDirConfig(v.(string), "group_vars"); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},

			"host_vars": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if err := validateDirConfig(v.(string), "host_vars"); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},

			"playbook_directory": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if err := validateDirConfig(v.(string), "playbook_directory"); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},

			"playbook_file": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if err := validateFileConfig(v.(string), "playbook_file"); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},

			"playbook_paths": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: func(v interface{}, k string) ([]string, []error) {
						if err := validateDirConfig(v.(string), "playbook_paths"); err != nil {
							return nil, []error{err}
						}
						return nil, nil
					},
				},
			},

			"role_paths": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: func(v interface{}, k string) ([]string, []error) {
						if err := validateDirConfig(v.(string), "role_paths"); err != nil {
							return nil, []error{err}
						}
						return nil, nil
					},
				},
			},

			"staging_directory": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: func() (interface{}, error) {
					id, err := uuid.GenerateUUID()
					if err != nil {
						return nil, err
					}
					return filepath.ToSlash(filepath.Join("/tmp/terraform-provisioner-ansible-local", id)), nil
				},
			},

			"inventory_file": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if err := validateFileConfig(v.(string), "inventory_file"); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
				ConflictsWith: []string{"inventory_groups"},
			},

			"inventory_groups": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"inventory_file"},
			},

			"galaxy_file": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if err := validateFileConfig(v.(string), "galaxy_file"); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},

			"galaxy_command": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ansible-galaxy",
			},
		},

		ApplyFunc: applyFn,
	}
}

func applyFn(ctx context.Context) error {
	connState := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	// Get a new communicator
	comm, err := communicator.New(connState)
	if err != nil {
		return err
	}

	return provisionWithAnsible(comm, data, o, func() (*os.File, error) {
		return ioutil.TempFile("", "terraform-provisioner-ansible-local")
	})

}

// Uploads the relevant files onto a remote machine and then executes Ansible locally to that machine
func provisionWithAnsible(comm communicator.Communicator, data *schema.ResourceData, o terraform.UIOutput,
	temporaryInventoryFile func() (*os.File, error)) error {

	o.Output("Provisioning with Ansible...")

	stagingDir := data.Get("staging_directory").(string)
	if playbookDir := data.Get("playbook_directory").(string); len(playbookDir) > 0 {
		o.Output("Uploading Playbook directory to Ansible staging directory...")
		if err := uploadDir(o, comm, stagingDir, playbookDir); err != nil {
			return fmt.Errorf("Error uploading playbook_dir directory: %s", err)
		}
	} else {
		o.Output("Creating Ansible staging directory...")
		if err := createDir(o, comm, stagingDir); err != nil {
			return fmt.Errorf("Error creating staging directory: %s", err)
		}
	}

	o.Output("Uploading main Playbook file...")
	playbookFile := data.Get("playbook_file").(string)
	src := playbookFile
	dst := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(src)))
	if err := uploadFile(comm, dst, src); err != nil {
		return fmt.Errorf("Error uploading main playbook: %s", err)
	}

	inventoryFile := data.Get("inventory_file").(string)

	if len(inventoryFile) == 0 {
		tf, err := temporaryInventoryFile()
		if err != nil {
			return fmt.Errorf("Error preparing inventory file: %s", err)
		}
		defer os.Remove(tf.Name())
		inventoryGroups := data.Get("inventory_groups").([]interface{})
		if len(inventoryGroups) != 0 {
			content := ""
			for _, group := range inventoryGroups {
				content += fmt.Sprintf("[%s]\n127.0.0.1\n", group)
			}
			_, err = tf.Write([]byte(content))
		} else {
			_, err = tf.Write([]byte("127.0.0.1"))
		}
		if err != nil {
			tf.Close()
			return fmt.Errorf("Error preparing inventory file: %s", err)
		}
		tf.Close()
		inventoryFile = tf.Name()
	}

	if galaxyFile := data.Get("galaxy_file").(string); len(galaxyFile) > 0 {
		o.Output("Uploading galaxy_file directory...")
		src := galaxyFile
		dst := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(src)))
		if err := uploadDir(o, comm, dst, src); err != nil {
			return fmt.Errorf("Error uploading galaxy file: %s", err)
		}
	}

	o.Output("Uploading inventory file...")
	src = inventoryFile
	dst = filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(src)))
	if err := uploadFile(comm, dst, src); err != nil {
		return fmt.Errorf("Error uploading inventory file: %s", err)
	}

	if groupVars := data.Get("group_vars").(string); len(groupVars) > 0 {
		o.Output("Uploading group_vars directory...")
		src := groupVars
		dst := filepath.ToSlash(filepath.Join(stagingDir, "group_vars"))
		if err := uploadDir(o, comm, dst, src); err != nil {
			return fmt.Errorf("Error uploading group_vars directory: %s", err)
		}
	}

	if hostVars := data.Get("host_vars").(string); len(hostVars) > 0 {
		o.Output("Uploading host_vars directory...")
		src := hostVars
		dst := filepath.ToSlash(filepath.Join(stagingDir, "host_vars"))
		if err := uploadDir(o, comm, dst, src); err != nil {
			return fmt.Errorf("Error uploading host_vars directory: %s", err)
		}
	}

	if rolePaths := data.Get("role_paths").([]interface{}); len(rolePaths) > 0 {
		o.Output("Uploading role directories...")
		for _, src := range rolePaths {
			dst := filepath.ToSlash(filepath.Join(stagingDir, "roles", filepath.Base(src.(string))))
			if err := uploadDir(o, comm, dst, src.(string)); err != nil {
				return fmt.Errorf("Error uploading roles: %s", err)
			}
		}
	}

	if playbookPaths := data.Get("playbook_paths").([]interface{}); len(playbookPaths) > 0 {
		o.Output("Uploading additional Playbooks...")
		playbookDir := filepath.ToSlash(filepath.Join(stagingDir, "playbooks"))
		if err := createDir(o, comm, playbookDir); err != nil {
			return fmt.Errorf("Error creating playbooks directory: %s", err)
		}
		for _, src := range playbookPaths {
			dst := filepath.ToSlash(filepath.Join(playbookDir, filepath.Base(src.(string))))
			if err := uploadDir(o, comm, dst, src.(string)); err != nil {
				return fmt.Errorf("Error uploading playbooks: %s", err)
			}
		}
	}

	if err := executeAnsible(data.Get("command").(string), stagingDir, playbookFile, inventoryFile,
		data.Get("galaxy_file").(string), data.Get("galaxy_command").(string),
		toStringArray(data.Get("extra_arguments").([]interface{})), o, comm); err != nil {
		return fmt.Errorf("Error executing Ansible: %s", err)
	}
	return nil
}

func executeGalaxy(stagingDir, galaxyFileStr, galaxyCommand string, o terraform.UIOutput, comm communicator.Communicator) error {
	rolesDir := filepath.ToSlash(filepath.Join(stagingDir, "roles"))
	galaxyFile := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(galaxyFileStr)))

	// ansible-galaxy install -r requirements.yml -p roles/
	command := fmt.Sprintf("cd %s && %s install -r %s -p %s",
		stagingDir, galaxyCommand, galaxyFile, rolesDir)
	o.Output(fmt.Sprintf("Executing Ansible Galaxy: %s", command))
	if err := runCommand(o, comm, command); err != nil {
		return err
	}

	return nil
}

func executeAnsible(ansibleCommand, stagingDir, playbookFile, inventoryFile, galaxyFile, galaxyCommand string,
	extraArguments []string, o terraform.UIOutput, comm communicator.Communicator) error {
	playbook := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(playbookFile)))
	inventory := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(inventoryFile)))

	extraArgs := " "
	if len(extraArguments) > 0 {
		extraArgs = extraArgs + strings.Join(extraArguments, " ")
	}

	if len(galaxyFile) > 0 {
		if err := executeGalaxy(stagingDir, galaxyFile, galaxyCommand, o, comm); err != nil {
			return fmt.Errorf("Error executing Ansible Galaxy: %s", err)
		}
	}

	command := fmt.Sprintf("cd %s && %s %s%s -c local -i %s",
		stagingDir, ansibleCommand, playbook, extraArgs, inventory)

	if err := runCommand(o, comm, command); err != nil {
		return err
	}

	return nil
}

func runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go copyOutput(o, outR, outDoneCh)
	go copyOutput(o, errR, errDoneCh)

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err := comm.Start(cmd)
	if err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	cmd.Wait()
	if cmd.ExitStatus != 0 {
		err = fmt.Errorf(
			"Command %q exited with non-zero exit status: %d", cmd.Command, cmd.ExitStatus)
	}

	// Wait for output to clean up
	outW.Close()
	errW.Close()
	<-outDoneCh
	<-errDoneCh

	return err
}

func validateDirConfig(path string, config string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: %s is invalid: %s", config, path, err)
	} else if !info.IsDir() {
		return fmt.Errorf("%s: %s must point to a directory", config, path)
	}
	return nil
}

func validateFileConfig(name string, config string) error {
	info, err := os.Stat(name)
	if err != nil {
		return fmt.Errorf("%s: %s is invalid: %s", config, name, err)
	} else if info.IsDir() {
		return fmt.Errorf("%s: %s must point to a file", config, name)
	}
	return nil
}

func uploadFile(comm communicator.Communicator, dst, src string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Error opening: %s", err)
	}
	defer f.Close()

	if err = comm.Upload(dst, f); err != nil {
		return fmt.Errorf("Error uploading %s: %s", src, err)
	}
	return nil
}

func createDir(o terraform.UIOutput, comm communicator.Communicator, dir string) error {
	o.Output(fmt.Sprintf("Creating directory: %s", dir))
	cmd := &remote.Cmd{
		Command: fmt.Sprintf("mkdir -p '%s'", dir),
	}

	if err := comm.Start(cmd); err != nil {
		return err
	}
	if cmd.ExitStatus != 0 {
		return fmt.Errorf("Non-zero exit status.")
	}
	return nil
}

func uploadDir(ui terraform.UIOutput, comm communicator.Communicator, dst, src string) error {
	if err := createDir(ui, comm, dst); err != nil {
		return err
	}

	// Make sure there is a trailing "/" so that the directory isn't
	// created on the other side.
	if src[len(src)-1] != '/' {
		src = src + "/"
	}
	return comm.UploadDir(dst, src)
}

func toStringArray(input []interface{}) []string {
	output := make([]string, len(input))

	for i, item := range input {
		output[i] = item.(string)
	}

	return output
}

func copyOutput(
	o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
