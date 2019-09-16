package puppet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

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
	Server            string
	ServerUser        string
	OSType            string
	Certname          string
	Environment       string
	Autosign          bool
	OpenSource        bool
	UseSudo           bool
	BoltTimeout       time.Duration
	CustomAttributes  map[string]interface{}
	ExtensionRequests map[string]interface{}

	runPuppetAgent     func() error
	installPuppetAgent func() error
	uploadFile         func(f io.Reader, dir string, filename string) error
	defaultCertname    func() (string, error)

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
				ValidateFunc: validation.StringInSlice([]string{"linux", "windows"}, false),
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
				Default:  true,
			},
			"certname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"extension_requests": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"custom_attributes": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"environment": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "production",
				Optional: true,
			},
			"bolt_timeout": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "5m",
				Optional: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					_, err := time.ParseDuration(val.(string))
					if err != nil {
						errs = append(errs, err)
					}
					return warns, errs
				},
			},
		},
		ApplyFunc: applyFn,
	}
}

func applyFn(ctx context.Context) error {
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
			p.OSType = "linux"
		case "winrm":
			p.OSType = "windows"
		default:
			return fmt.Errorf("Unsupported connection type: %s", connType)
		}
	}

	switch p.OSType {
	case "linux":
		p.runPuppetAgent = p.linuxRunPuppetAgent
		p.installPuppetAgent = p.linuxInstallPuppetAgent
		p.uploadFile = p.linuxUploadFile
		p.defaultCertname = p.linuxDefaultCertname
	case "windows":
		p.runPuppetAgent = p.windowsRunPuppetAgent
		p.installPuppetAgent = p.windowsInstallPuppetAgent
		p.uploadFile = p.windowsUploadFile
		p.UseSudo = false
		p.defaultCertname = p.windowsDefaultCertname
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
	defer comm.Disconnect()

	p.comm = comm

	if p.OpenSource {
		p.installPuppetAgent = p.installPuppetAgentOpenSource
	}

	csrAttrs := new(csrAttributes)
	csrAttrs.CustomAttributes = make(map[string]string)
	for k, v := range p.CustomAttributes {
		csrAttrs.CustomAttributes[k] = v.(string)
	}

	csrAttrs.ExtensionRequests = make(map[string]string)
	for k, v := range p.ExtensionRequests {
		csrAttrs.ExtensionRequests[k] = v.(string)
	}

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

func (p *provisioner) writeCSRAttributes(attrs *csrAttributes) (rerr error) {
	content, err := yaml.Marshal(attrs)
	if err != nil {
		return fmt.Errorf("Failed to marshal CSR attributes to YAML: %s", err)
	}

	configDir := map[string]string{
		"linux":   "/etc/puppetlabs/puppet",
		"windows": "C:\\ProgramData\\PuppetLabs\\Puppet\\etc",
	}

	return p.uploadFile(bytes.NewBuffer(content), configDir[p.OSType], "csr_attributes.yaml")
}

func (p *provisioner) generateAutosignToken(certname string) (string, error) {
	task := "autosign::generate_token"

	masterConnInfo := map[string]string{
		"type": "ssh",
		"host": p.Server,
		"user": p.ServerUser,
	}

	result, err := bolt.Task(
		masterConnInfo,
		p.BoltTimeout,
		p.ServerUser != "root",
		task,
		map[string]string{"certname": certname},
	)
	if err != nil {
		return "", err
	}

	if result.Items[0].Status != "success" {
		return "", fmt.Errorf("Bolt %s failed on %s: %v",
			task,
			result.Items[0].Node,
			result.Items[0].Result["_error"],
		)
	}

	return result.Items[0].Result["_output"], nil
}

func (p *provisioner) installPuppetAgentOpenSource() error {
	result, err := bolt.Task(
		p.instanceState.Ephemeral.ConnInfo,
		p.BoltTimeout,
		p.UseSudo,
		"puppet_agent::install",
		nil,
	)

	if err != nil || result.Items[0].Status != "success" {
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
		return stdout, err
	}

	err = cmd.Wait()
	stdout = strings.TrimSpace(stdoutBuffer.String())

	return stdout, err
}

func (p *provisioner) copyToOutput(reader io.Reader) {
	lr := linereader.New(reader)
	for line := range lr.Ch {
		p.output.Output(line)
	}
}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		UseSudo:           d.Get("use_sudo").(bool),
		Server:            d.Get("server").(string),
		ServerUser:        d.Get("server_user").(string),
		OSType:            strings.ToLower(d.Get("os_type").(string)),
		Autosign:          d.Get("autosign").(bool),
		OpenSource:        d.Get("open_source").(bool),
		Certname:          strings.ToLower(d.Get("certname").(string)),
		ExtensionRequests: d.Get("extension_requests").(map[string]interface{}),
		CustomAttributes:  d.Get("custom_attributes").(map[string]interface{}),
		Environment:       d.Get("environment").(string),
	}
	p.BoltTimeout, _ = time.ParseDuration(d.Get("bolt_timeout").(string))

	return p, nil
}
