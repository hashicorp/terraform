// This package implements a provisioner for Terraform that executes a
// saltstack state within the remote machine
//
// Adapted from gitub.com/hashicorp/packer/provisioner/salt-masterless

package saltmasterless

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	linereader "github.com/mitchellh/go-linereader"
)

type provisionFn func(terraform.UIOutput, communicator.Communicator) error

type provisioner struct {
	SkipBootstrap     bool
	BootstrapArgs     string
	LocalStateTree    string
	DisableSudo       bool
	CustomState       string
	MinionConfig      string
	LocalPillarRoots  string
	RemoteStateTree   string
	RemotePillarRoots string
	TempConfigDir     string
	NoExitOnFailure   bool
	LogLevel          string
	SaltCallArgs      string
	CmdArgs           string
}

const DefaultStateTreeDir = "/srv/salt"
const DefaultPillarRootDir = "/srv/pillar"

// Provisioner returns a salt-masterless provisioner
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"local_state_tree": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"local_pillar_roots": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"remote_state_tree": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"remote_pillar_roots": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"temp_config_dir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/tmp/salt",
			},
			"skip_bootstrap": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"no_exit_on_failure": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"bootstrap_args": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"disable_sudo": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"custom_state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"minion_config_file": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cmd_args": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"salt_call_args": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"log_level": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		ApplyFunc:    applyFn,
		ValidateFunc: validateFn,
	}
}

// Apply executes the file provisioner
func applyFn(ctx context.Context) error {
	// Decode the raw config for this provisioner
	var err error

	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	connState := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)

	p, err := decodeConfig(d)
	if err != nil {
		return err
	}

	// Get a new communicator
	comm, err := communicator.New(connState)
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithTimeout(ctx, comm.Timeout())
	defer cancelFunc()

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	// Wait and retry until we establish the connection
	err = communicator.Retry(ctx, func() error {
		return comm.Connect(o)
	})

	if err != nil {
		return err
	}

	var src, dst string

	o.Output("Provisioning with Salt...")
	if !p.SkipBootstrap {
		cmd := &remote.Cmd{
			// Fallback on wget if curl failed for any reason (such as not being installed)
			Command: fmt.Sprintf("curl -L https://bootstrap.saltstack.com -o /tmp/install_salt.sh || wget -O /tmp/install_salt.sh https://bootstrap.saltstack.com"),
		}
		o.Output(fmt.Sprintf("Downloading saltstack bootstrap to /tmp/install_salt.sh"))
		if err = comm.Start(cmd); err != nil {
			err = fmt.Errorf("Unable to download Salt: %s", err)
		}

		if err == nil {
			cmd.Wait()
			if cmd.ExitStatus != 0 {
				err = fmt.Errorf("Curl exited with non-zero exit status: %d", cmd.ExitStatus)
			}
		}

		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		outDoneCh := make(chan struct{})
		errDoneCh := make(chan struct{})
		go copyOutput(o, outR, outDoneCh)
		go copyOutput(o, errR, errDoneCh)
		cmd = &remote.Cmd{
			Command: fmt.Sprintf("%s /tmp/install_salt.sh %s", p.sudo("sh"), p.BootstrapArgs),
			Stdout:  outW,
			Stderr:  errW,
		}

		o.Output(fmt.Sprintf("Installing Salt with command %s", cmd.Command))
		if err = comm.Start(cmd); err != nil {
			err = fmt.Errorf("Unable to install Salt: %s", err)
		}

		if err == nil {
			cmd.Wait()
			if cmd.ExitStatus != 0 {
				err = fmt.Errorf("install_salt.sh exited with non-zero exit status: %d", cmd.ExitStatus)
			}
		}
		// Wait for output to clean up
		outW.Close()
		errW.Close()
		<-outDoneCh
		<-errDoneCh
		if err != nil {
			return err
		}
	}

	o.Output(fmt.Sprintf("Creating remote temporary directory: %s", p.TempConfigDir))
	if err := p.createDir(o, comm, p.TempConfigDir); err != nil {
		return fmt.Errorf("Error creating remote temporary directory: %s", err)
	}

	if p.MinionConfig != "" {
		o.Output(fmt.Sprintf("Uploading minion config: %s", p.MinionConfig))
		src = p.MinionConfig
		dst = filepath.ToSlash(filepath.Join(p.TempConfigDir, "minion"))
		if err = p.uploadFile(o, comm, dst, src); err != nil {
			return fmt.Errorf("Error uploading local minion config file to remote: %s", err)
		}

		// move minion config into /etc/salt
		o.Output(fmt.Sprintf("Make sure directory %s exists", "/etc/salt"))
		if err := p.createDir(o, comm, "/etc/salt"); err != nil {
			return fmt.Errorf("Error creating remote salt configuration directory: %s", err)
		}
		src = filepath.ToSlash(filepath.Join(p.TempConfigDir, "minion"))
		dst = "/etc/salt/minion"
		if err = p.moveFile(o, comm, dst, src); err != nil {
			return fmt.Errorf("Unable to move %s/minion to /etc/salt/minion: %s", p.TempConfigDir, err)
		}
	}

	o.Output(fmt.Sprintf("Uploading local state tree: %s", p.LocalStateTree))
	src = p.LocalStateTree
	dst = filepath.ToSlash(filepath.Join(p.TempConfigDir, "states"))
	if err = p.uploadDir(o, comm, dst, src, []string{".git"}); err != nil {
		return fmt.Errorf("Error uploading local state tree to remote: %s", err)
	}

	// move state tree from temporary directory
	src = filepath.ToSlash(filepath.Join(p.TempConfigDir, "states"))
	dst = p.RemoteStateTree
	if err = p.removeDir(o, comm, dst); err != nil {
		return fmt.Errorf("Unable to clear salt tree: %s", err)
	}
	if err = p.moveFile(o, comm, dst, src); err != nil {
		return fmt.Errorf("Unable to move %s/states to %s: %s", p.TempConfigDir, dst, err)
	}

	if p.LocalPillarRoots != "" {
		o.Output(fmt.Sprintf("Uploading local pillar roots: %s", p.LocalPillarRoots))
		src = p.LocalPillarRoots
		dst = filepath.ToSlash(filepath.Join(p.TempConfigDir, "pillar"))
		if err = p.uploadDir(o, comm, dst, src, []string{".git"}); err != nil {
			return fmt.Errorf("Error uploading local pillar roots to remote: %s", err)
		}

		// move pillar root from temporary directory
		src = filepath.ToSlash(filepath.Join(p.TempConfigDir, "pillar"))
		dst = p.RemotePillarRoots

		if err = p.removeDir(o, comm, dst); err != nil {
			return fmt.Errorf("Unable to clear pillar root: %s", err)
		}
		if err = p.moveFile(o, comm, dst, src); err != nil {
			return fmt.Errorf("Unable to move %s/pillar to %s: %s", p.TempConfigDir, dst, err)
		}
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})

	go copyOutput(o, outR, outDoneCh)
	go copyOutput(o, errR, errDoneCh)
	o.Output(fmt.Sprintf("Running: salt-call --local %s", p.CmdArgs))
	cmd := &remote.Cmd{
		Command: p.sudo(fmt.Sprintf("salt-call --local %s", p.CmdArgs)),
		Stdout:  outW,
		Stderr:  errW,
	}
	if err = comm.Start(cmd); err != nil || cmd.ExitStatus != 0 {
		if err == nil {
			err = fmt.Errorf("Bad exit status: %d", cmd.ExitStatus)
		}

		err = fmt.Errorf("Error executing salt-call: %s", err)
	}
	if err == nil {
		cmd.Wait()
		if cmd.ExitStatus != 0 {
			err = fmt.Errorf("Script exited with non-zero exit status: %d", cmd.ExitStatus)
		}
	}
	// Wait for output to clean up
	outW.Close()
	errW.Close()
	<-outDoneCh
	<-errDoneCh

	return err
}

// Prepends sudo to supplied command if config says to
func (p *provisioner) sudo(cmd string) string {
	if p.DisableSudo {
		return cmd
	}

	return "sudo " + cmd
}

func validateDirConfig(path string, name string, required bool) error {
	if required == true && path == "" {
		return fmt.Errorf("%s cannot be empty", name)
	} else if required == false && path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: path '%s' is invalid: %s", name, path, err)
	} else if !info.IsDir() {
		return fmt.Errorf("%s: path '%s' must point to a directory", name, path)
	}
	return nil
}

func validateFileConfig(path string, name string, required bool) error {
	if required == true && path == "" {
		return fmt.Errorf("%s cannot be empty", name)
	} else if required == false && path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: path '%s' is invalid: %s", name, path, err)
	} else if info.IsDir() {
		return fmt.Errorf("%s: path '%s' must point to a file", name, path)
	}
	return nil
}

func (p *provisioner) uploadFile(o terraform.UIOutput, comm communicator.Communicator, dst, src string) error {
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

func (p *provisioner) moveFile(o terraform.UIOutput, comm communicator.Communicator, dst, src string) error {
	o.Output(fmt.Sprintf("Moving %s to %s", src, dst))
	cmd := &remote.Cmd{Command: fmt.Sprintf(p.sudo("mv %s %s"), src, dst)}
	if err := comm.Start(cmd); err != nil || cmd.ExitStatus != 0 {
		if err == nil {
			err = fmt.Errorf("Bad exit status: %d", cmd.ExitStatus)
		}

		return fmt.Errorf("Unable to move %s to %s: %s", src, dst, err)
	}
	return nil
}

func (p *provisioner) createDir(o terraform.UIOutput, comm communicator.Communicator, dir string) error {
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

func (p *provisioner) removeDir(o terraform.UIOutput, comm communicator.Communicator, dir string) error {
	o.Output(fmt.Sprintf("Removing directory: %s", dir))
	cmd := &remote.Cmd{
		Command: fmt.Sprintf("rm -rf '%s'", dir),
	}
	if err := comm.Start(cmd); err != nil {
		return err
	}
	if cmd.ExitStatus != 0 {
		return fmt.Errorf("Non-zero exit status.")
	}
	return nil
}

func (p *provisioner) uploadDir(o terraform.UIOutput, comm communicator.Communicator, dst, src string, ignore []string) error {
	if err := p.createDir(o, comm, dst); err != nil {
		return err
	}

	// Make sure there is a trailing "/" so that the directory isn't
	// created on the other side.
	if src[len(src)-1] != '/' {
		src = src + "/"
	}
	return comm.UploadDir(dst, src)
}

// Validate checks if the required arguments are configured
func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
	// require a salt state tree
	localStateTreeTmp, ok := c.Get("local_state_tree")
	var localStateTree string
	if !ok {
		es = append(es,
			errors.New("Required local_state_tree is not set"))
	} else {
		localStateTree = localStateTreeTmp.(string)
	}
	err := validateDirConfig(localStateTree, "local_state_tree", true)
	if err != nil {
		es = append(es, err)
	}

	var localPillarRoots string
	localPillarRootsTmp, ok := c.Get("local_pillar_roots")
	if !ok {
		localPillarRoots = ""
	} else {
		localPillarRoots = localPillarRootsTmp.(string)
	}

	err = validateDirConfig(localPillarRoots, "local_pillar_roots", false)
	if err != nil {
		es = append(es, err)
	}

	var minionConfig string
	minionConfigTmp, ok := c.Get("minion_config_file")
	if !ok {
		minionConfig = ""
	} else {
		minionConfig = minionConfigTmp.(string)
	}
	err = validateFileConfig(minionConfig, "minion_config_file", false)
	if err != nil {
		es = append(es, err)
	}

	var remoteStateTree string
	remoteStateTreeTmp, ok := c.Get("remote_state_tree")
	if !ok {
		remoteStateTree = ""
	} else {
		remoteStateTree = remoteStateTreeTmp.(string)
	}

	var remotePillarRoots string
	remotePillarRootsTmp, ok := c.Get("remote_pillar_roots")
	if !ok {
		remotePillarRoots = ""
	} else {
		remotePillarRoots = remotePillarRootsTmp.(string)
	}

	if minionConfig != "" && (remoteStateTree != "" || remotePillarRoots != "") {
		es = append(es,
			errors.New("remote_state_tree and remote_pillar_roots only apply when minion_config_file is not used"))
	}

	if len(es) > 0 {
		return ws, es
	}

	return ws, es
}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		LocalStateTree:    d.Get("local_state_tree").(string),
		LogLevel:          d.Get("log_level").(string),
		SaltCallArgs:      d.Get("salt_call_args").(string),
		CmdArgs:           d.Get("cmd_args").(string),
		MinionConfig:      d.Get("minion_config_file").(string),
		CustomState:       d.Get("custom_state").(string),
		DisableSudo:       d.Get("disable_sudo").(bool),
		BootstrapArgs:     d.Get("bootstrap_args").(string),
		NoExitOnFailure:   d.Get("no_exit_on_failure").(bool),
		SkipBootstrap:     d.Get("skip_bootstrap").(bool),
		TempConfigDir:     d.Get("temp_config_dir").(string),
		RemotePillarRoots: d.Get("remote_pillar_roots").(string),
		RemoteStateTree:   d.Get("remote_state_tree").(string),
		LocalPillarRoots:  d.Get("local_pillar_roots").(string),
	}

	// build the command line args to pass onto salt
	var cmdArgs bytes.Buffer

	if p.CustomState == "" {
		cmdArgs.WriteString(" state.highstate")
	} else {
		cmdArgs.WriteString(" state.sls ")
		cmdArgs.WriteString(p.CustomState)
	}

	if p.MinionConfig == "" {
		// pass --file-root and --pillar-root if no minion_config_file is supplied
		if p.RemoteStateTree != "" {
			cmdArgs.WriteString(" --file-root=")
			cmdArgs.WriteString(p.RemoteStateTree)
		} else {
			cmdArgs.WriteString(" --file-root=")
			cmdArgs.WriteString(DefaultStateTreeDir)
		}
		if p.RemotePillarRoots != "" {
			cmdArgs.WriteString(" --pillar-root=")
			cmdArgs.WriteString(p.RemotePillarRoots)
		} else {
			cmdArgs.WriteString(" --pillar-root=")
			cmdArgs.WriteString(DefaultPillarRootDir)
		}
	}

	if !p.NoExitOnFailure {
		cmdArgs.WriteString(" --retcode-passthrough")
	}

	if p.LogLevel == "" {
		cmdArgs.WriteString(" -l info")
	} else {
		cmdArgs.WriteString(" -l ")
		cmdArgs.WriteString(p.LogLevel)
	}

	if p.SaltCallArgs != "" {
		cmdArgs.WriteString(" ")
		cmdArgs.WriteString(p.SaltCallArgs)
	}

	p.CmdArgs = cmdArgs.String()

	return p, nil
}

func copyOutput(
	o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
