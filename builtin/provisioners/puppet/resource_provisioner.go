package puppet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/terraform/builtin/provisioners/puppet/bolt"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
	"gopkg.in/yaml.v2"
)

type provisioner struct {
	Server      string
	ServerUser  string
	OSType      string
	Certname    string
	PPRole      string
	Environment string
	Autosign    bool
	OpenSource  bool
	UseSudo     bool

	runPuppetAgent     func() error
	installPuppetAgent func() error
	uploadFile         func(f io.Reader, dir string, filename string) error
	defaultCertname    func() (string, error)
	isPuppetEnterprise func() (bool, error)

	instanceState *terraform.InstanceState
	output        terraform.UIOutput
	comm          communicator.Communicator
}

type csrAttributes struct {
	CustomAttributes  map[string]string `yaml:"custom_attributes"`
	ExtensionRequests map[string]string `yaml:"extension_requests"`
}

// Provisioner returns a Puppet resource provisioner.
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"server": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"server_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "root",
			},
			"os_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"nix", "windows"}, false),
			},
			"use_sudo": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"autosign": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"open_source": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"certname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"pp_role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "production",
				Optional: true,
			},
		},
		ApplyFunc:    applyFn,
		ValidateFunc: validateFn,
	}
}

func applyFn(ctx context.Context) (rerr error) {
	output := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
	state := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	configData := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	p, err := decodeConfig(configData)
	if err != nil {
		return err
	}

	p.instanceState = state
	p.output = output

	if p.OSType == "" {
		switch connType := state.Ephemeral.ConnInfo["type"]; connType {
		case "ssh", "":
			p.OSType = "nix"
		case "winrm":
			p.OSType = "windows"
		default:
			return fmt.Errorf("Unsupported connection type: %s", connType)
		}
	}

	switch p.OSType {
	case "nix":
		p.runPuppetAgent = p.nixRunPuppetAgent
		p.installPuppetAgent = p.nixInstallPuppetAgent
		p.uploadFile = p.nixUploadFile
		p.defaultCertname = p.nixDefaultCertname
		p.isPuppetEnterprise = p.nixIsPuppetEnterprise
	case "windows":
		p.runPuppetAgent = p.windowsRunPuppetAgent
		p.installPuppetAgent = p.windowsInstallPuppetAgent
		p.uploadFile = p.windowsUploadFile
		p.UseSudo = false
		p.defaultCertname = p.windowsDefaultCertname
		p.isPuppetEnterprise = p.windowsIsPuppetEnterprise
	default:
		return fmt.Errorf("Unsupported OS type: %s", p.OSType)
	}

	comm, err := communicator.New(state)
	if err != nil {
		return err
	}

	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	err = communicator.Retry(retryCtx, func() error {
		return comm.Connect(output)
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := comm.Disconnect(); err != nil {
			rerr = err
		}
	}()

	p.comm = comm

	_, ok := configData.GetOkExists("open_source")
	if !ok {
		isPE, err := p.isPuppetEnterprise()
		if err != nil {
			return fmt.Errorf("Unable to determine if %s is running Puppet Enterprise: %v", p.Server, err)
		}
		p.OpenSource = !isPE
	}

	if p.OpenSource {
		p.installPuppetAgent = p.installPuppetAgentOpenSource
	}

	csrAttrs := new(csrAttributes)
	csrAttrs.CustomAttributes = make(map[string]string)
	csrAttrs.ExtensionRequests = make(map[string]string)

	if p.Autosign {
		if p.Certname == "" {
			p.Certname, _ = p.defaultCertname()
		}

		autosignToken, err := p.generateAutosignToken(p.Certname)
		if err != nil {
			return fmt.Errorf("Failed to generate an autosign token: %s", err)
		}
		csrAttrs.CustomAttributes["challengePassword"] = autosignToken
	}

	if p.PPRole != "" {
		csrAttrs.ExtensionRequests["pp_role"] = p.PPRole
	}

	if err = p.writeCSRAttributes(csrAttrs); err != nil {
		return fmt.Errorf("Failed to write csr_attributes.yaml: %s", err)
	}

	if err = p.installPuppetAgent(); err != nil {
		return err
	}

	if err = p.runPuppetAgent(); err != nil {
		return err
	}

	return nil
}

func validateFn(config *terraform.ResourceConfig) (ws []string, es []error) {
	return ws, es
}

func (p *provisioner) writeCSRAttributes(attrs *csrAttributes) (rerr error) {
	file, err := ioutil.TempFile("", "puppet-csr-attrs")
	if err != nil {
		return fmt.Errorf("Failed to create a temp file: %s", err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			rerr = err
		}
	}()

	content, err := yaml.Marshal(attrs)
	if err != nil {
		return fmt.Errorf("Failed to marshal CSR attributes to YAML: %s", err)
	}

	_, err = file.WriteString(string(content))
	if err != nil {
		return fmt.Errorf("Failed to write YAML to temp file: %s", err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	configDir := map[string]string{
		"nix":     "/etc/puppetlabs/puppet",
		"windows": "C:\\ProgramData\\PuppetLabs\\Puppet\\etc",
	}

	return p.uploadFile(file, configDir[p.OSType], "csr_attributes.yaml")
}

func (p *provisioner) generateAutosignToken(certname string) (string, error) {
	masterConnInfo := map[string]string{
		"type": "ssh",
		"host": p.Server,
		"user": p.ServerUser,
	}

	result, err := bolt.Task(
		masterConnInfo,
		p.ServerUser != "root",
		"autosign::generate_token",
		map[string]string{"certname": certname},
	)
	if err != nil {
		return "", err
	}
	// TODO check error state in JSON
	return result.Items[0].Result["_output"], nil
}

func (p *provisioner) installPuppetAgentOpenSource() error {
	result, err := bolt.Task(
		p.instanceState.Ephemeral.ConnInfo,
		p.UseSudo,
		"puppet_agent::install",
		nil,
	)

	if err != nil {
		return fmt.Errorf("puppet_agent::install failed: %s\n%+v", err, result)
	}

	return nil
}

func (p *provisioner) runCommand(command string) (stdout string, err error) {
	if p.UseSudo {
		command = "sudo " + command
	}

	var stdoutBuffer bytes.Buffer
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outTee := io.TeeReader(outR, &stdoutBuffer)
	go p.copyToOutput(outTee)
	go p.copyToOutput(errR)
	defer outW.Close()
	defer errW.Close()

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err = p.comm.Start(cmd)
	if err != nil {
		err = fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
		return
	}

	err = cmd.Wait()
	stdout = strings.TrimSpace(stdoutBuffer.String())

	return
}

func (p *provisioner) copyToOutput(reader io.Reader) {
	lr := linereader.New(reader)
	for line := range lr.Ch {
		p.output.Output(line)
	}
}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		UseSudo:     d.Get("use_sudo").(bool),
		Server:      d.Get("server").(string),
		ServerUser:  d.Get("server_user").(string),
		OSType:      strings.ToLower(d.Get("os_type").(string)),
		Autosign:    d.Get("autosign").(bool),
		OpenSource:  d.Get("open_source").(bool),
		Certname:    strings.ToLower(d.Get("certname").(string)),
		PPRole:      d.Get("pp_role").(string),
		Environment: d.Get("environment").(string),
	}

	return p, nil
}
