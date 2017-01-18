package chef

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-linereader"
	"github.com/mitchellh/mapstructure"
)

const (
	clienrb         = "client.rb"
	defaultEnv      = "_default"
	firstBoot       = "first-boot.json"
	logfileDir      = "logfiles"
	linuxChefCmd    = "chef-client"
	linuxConfDir    = "/etc/chef"
	linuxNoOutput   = "> /dev/null 2>&1"
	linuxGemCmd     = "/opt/chef/embedded/bin/gem"
	linuxKnifeCmd   = "knife"
	secretKey       = "encrypted_data_bag_secret"
	windowsChefCmd  = "cmd /c chef-client"
	windowsConfDir  = "C:/chef"
	windowsNoOutput = "> nul 2>&1"
	windowsGemCmd   = "C:/opscode/chef/embedded/bin/gem"
	windowsKnifeCmd = "cmd /c knife"
)

const clientConf = `
log_location            STDOUT
chef_server_url         "{{ .ServerURL }}"
node_name               "{{ .NodeName }}"
{{ if .UsePolicyfile }}
use_policyfile true
policy_group 	 "{{ .PolicyGroup }}"
policy_name 	 "{{ .PolicyName }}"
{{ end -}}

{{ if .HTTPProxy }}
http_proxy          "{{ .HTTPProxy }}"
ENV['http_proxy'] = "{{ .HTTPProxy }}"
ENV['HTTP_PROXY'] = "{{ .HTTPProxy }}"
{{ end -}}

{{ if .HTTPSProxy }}
https_proxy          "{{ .HTTPSProxy }}"
ENV['https_proxy'] = "{{ .HTTPSProxy }}"
ENV['HTTPS_PROXY'] = "{{ .HTTPSProxy }}"
{{ end -}}

{{ if .NOProxy }}
no_proxy          "{{ join .NOProxy "," }}"
ENV['no_proxy'] = "{{ join .NOProxy "," }}"
{{ end -}}

{{ if .SSLVerifyMode }}
ssl_verify_mode  {{ .SSLVerifyMode }}
{{- end -}}

{{ if .DisableReporting }}
enable_reporting false
{{ end -}}

{{ if .ClientOptions }}
{{ join .ClientOptions "\n" }}
{{ end }}
`

// Provisioner represents a Chef provisioner
type Provisioner struct {
	AttributesJSON        string   `mapstructure:"attributes_json"`
	ClientOptions         []string `mapstructure:"client_options"`
	DisableReporting      bool     `mapstructure:"disable_reporting"`
	Environment           string   `mapstructure:"environment"`
	FetchChefCertificates bool     `mapstructure:"fetch_chef_certificates"`
	LogToFile             bool     `mapstructure:"log_to_file"`
	UsePolicyfile         bool     `mapstructure:"use_policyfile"`
	PolicyGroup           string   `mapstructure:"policy_group"`
	PolicyName            string   `mapstructure:"policy_name"`
	HTTPProxy             string   `mapstructure:"http_proxy"`
	HTTPSProxy            string   `mapstructure:"https_proxy"`
	NamedRunList          string   `mapstructure:"named_run_list"`
	NOProxy               []string `mapstructure:"no_proxy"`
	NodeName              string   `mapstructure:"node_name"`
	OhaiHints             []string `mapstructure:"ohai_hints"`
	OSType                string   `mapstructure:"os_type"`
	RecreateClient        bool     `mapstructure:"recreate_client"`
	PreventSudo           bool     `mapstructure:"prevent_sudo"`
	RunList               []string `mapstructure:"run_list"`
	SecretKey             string   `mapstructure:"secret_key"`
	ServerURL             string   `mapstructure:"server_url"`
	SkipInstall           bool     `mapstructure:"skip_install"`
	SkipRegister          bool     `mapstructure:"skip_register"`
	SSLVerifyMode         string   `mapstructure:"ssl_verify_mode"`
	UserName              string   `mapstructure:"user_name"`
	UserKey               string   `mapstructure:"user_key"`
	VaultJSON             string   `mapstructure:"vault_json"`
	Version               string   `mapstructure:"version"`

	attributes map[string]interface{}
	vaults     map[string][]string

	cleanupUserKeyCmd     string
	createConfigFiles     func(terraform.UIOutput, communicator.Communicator) error
	installChefClient     func(terraform.UIOutput, communicator.Communicator) error
	fetchChefCertificates func(terraform.UIOutput, communicator.Communicator) error
	generateClientKey     func(terraform.UIOutput, communicator.Communicator) error
	configureVaults       func(terraform.UIOutput, communicator.Communicator) error
	runChefClient         func(terraform.UIOutput, communicator.Communicator) error
	useSudo               bool

	// Deprecated Fields
	ValidationClientName string `mapstructure:"validation_client_name"`
	ValidationKey        string `mapstructure:"validation_key"`
}

// ResourceProvisioner represents a generic chef provisioner
type ResourceProvisioner struct{}

// Apply executes the file provisioner
func (r *ResourceProvisioner) Apply(
	o terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	// Decode the raw config for this provisioner
	p, err := r.decodeConfig(c)
	if err != nil {
		return err
	}

	if p.OSType == "" {
		switch s.Ephemeral.ConnInfo["type"] {
		case "ssh", "": // The default connection type is ssh, so if the type is empty assume ssh
			p.OSType = "linux"
		case "winrm":
			p.OSType = "windows"
		default:
			return fmt.Errorf("Unsupported connection type: %s", s.Ephemeral.ConnInfo["type"])
		}
	}

	// Set some values based on the targeted OS
	switch p.OSType {
	case "linux":
		p.cleanupUserKeyCmd = fmt.Sprintf("rm -f %s", path.Join(linuxConfDir, p.UserName+".pem"))
		p.createConfigFiles = p.linuxCreateConfigFiles
		p.installChefClient = p.linuxInstallChefClient
		p.fetchChefCertificates = p.fetchChefCertificatesFunc(linuxKnifeCmd, linuxConfDir)
		p.generateClientKey = p.generateClientKeyFunc(linuxKnifeCmd, linuxConfDir, linuxNoOutput)
		p.configureVaults = p.configureVaultsFunc(linuxGemCmd, linuxKnifeCmd, linuxConfDir)
		p.runChefClient = p.runChefClientFunc(linuxChefCmd, linuxConfDir)
		p.useSudo = !p.PreventSudo && s.Ephemeral.ConnInfo["user"] != "root"
	case "windows":
		p.cleanupUserKeyCmd = fmt.Sprintf("cd %s && del /F /Q %s", windowsConfDir, p.UserName+".pem")
		p.createConfigFiles = p.windowsCreateConfigFiles
		p.installChefClient = p.windowsInstallChefClient
		p.fetchChefCertificates = p.fetchChefCertificatesFunc(windowsKnifeCmd, windowsConfDir)
		p.generateClientKey = p.generateClientKeyFunc(windowsKnifeCmd, windowsConfDir, windowsNoOutput)
		p.configureVaults = p.configureVaultsFunc(windowsGemCmd, windowsKnifeCmd, windowsConfDir)
		p.runChefClient = p.runChefClientFunc(windowsChefCmd, windowsConfDir)
		p.useSudo = false
	default:
		return fmt.Errorf("Unsupported os type: %s", p.OSType)
	}

	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return err
	}

	// Wait and retry until we establish the connection
	err = retryFunc(comm.Timeout(), func() error {
		err := comm.Connect(o)
		return err
	})
	if err != nil {
		return err
	}
	defer comm.Disconnect()

	// Make sure we always delete the user key from the new node!
	var once sync.Once
	cleanupUserKey := func() {
		o.Output("Cleanup user key...")
		if err := p.runCommand(o, comm, p.cleanupUserKeyCmd); err != nil {
			o.Output("WARNING: Failed to cleanup user key on new node: " + err.Error())
		}
	}
	defer once.Do(cleanupUserKey)

	if !p.SkipInstall {
		if err := p.installChefClient(o, comm); err != nil {
			return err
		}
	}

	o.Output("Creating configuration files...")
	if err := p.createConfigFiles(o, comm); err != nil {
		return err
	}

	if !p.SkipRegister {
		if p.FetchChefCertificates {
			o.Output("Fetch Chef certificates...")
			if err := p.fetchChefCertificates(o, comm); err != nil {
				return err
			}
		}

		o.Output("Generate the private key...")
		if err := p.generateClientKey(o, comm); err != nil {
			return err
		}
	}

	if p.VaultJSON != "" {
		o.Output("Configure Chef vaults...")
		if err := p.configureVaults(o, comm); err != nil {
			return err
		}
	}

	// Cleanup the user key before we run Chef-Client to prevent issues
	// with rights caused by changing settings during the run.
	once.Do(cleanupUserKey)

	o.Output("Starting initial Chef-Client run...")
	if err := p.runChefClient(o, comm); err != nil {
		return err
	}

	return nil
}

// Validate checks if the required arguments are configured
func (r *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	p, err := r.decodeConfig(c)
	if err != nil {
		es = append(es, err)
		return ws, es
	}

	if p.NodeName == "" {
		es = append(es, errors.New("Key not found: node_name"))
	}
	if !p.UsePolicyfile && p.RunList == nil {
		es = append(es, errors.New("Key not found: run_list"))
	}
	if p.ServerURL == "" {
		es = append(es, errors.New("Key not found: server_url"))
	}
	if p.UsePolicyfile && p.PolicyName == "" {
		es = append(es, errors.New("Policyfile enabled but key not found: policy_name"))
	}
	if p.UsePolicyfile && p.PolicyGroup == "" {
		es = append(es, errors.New("Policyfile enabled but key not found: policy_group"))
	}
	if p.UserName == "" && p.ValidationClientName == "" {
		es = append(es, errors.New(
			"One of user_name or the deprecated validation_client_name must be provided"))
	}
	if p.UserKey == "" && p.ValidationKey == "" {
		es = append(es, errors.New(
			"One of user_key or the deprecated validation_key must be provided"))
	}
	if p.ValidationClientName != "" {
		ws = append(ws, "validation_client_name is deprecated, please use user_name instead")
	}
	if p.ValidationKey != "" {
		ws = append(ws, "validation_key is deprecated, please use user_key instead")

		if p.RecreateClient {
			es = append(es, errors.New(
				"Cannot use recreate_client=true with the deprecated validation_key, please provide a user_key"))
		}
		if p.VaultJSON != "" {
			es = append(es, errors.New(
				"Cannot configure chef vaults using the deprecated validation_key, please provide a user_key"))
		}
	}

	return ws, es
}

func (r *ResourceProvisioner) decodeConfig(c *terraform.ResourceConfig) (*Provisioner, error) {
	p := new(Provisioner)

	decConf := &mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           p,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}

	// We need to merge both configs into a single map first. Order is
	// important as we need to make sure interpolated values are used
	// over raw values. This makes sure that all values are there even
	// if some still need to be interpolated later on. Without this
	// the validation will fail when using a variable for a required
	// parameter (the node_name for example).
	m := make(map[string]interface{})

	for k, v := range c.Raw {
		m[k] = v
	}

	for k, v := range c.Config {
		m[k] = v
	}

	if err := dec.Decode(m); err != nil {
		return nil, err
	}

	// Make sure the supplied URL has a trailing slash
	p.ServerURL = strings.TrimSuffix(p.ServerURL, "/") + "/"

	if p.Environment == "" {
		p.Environment = defaultEnv
	}

	for i, hint := range p.OhaiHints {
		hintPath, err := homedir.Expand(hint)
		if err != nil {
			return nil, fmt.Errorf("Error expanding the path %s: %v", hint, err)
		}
		p.OhaiHints[i] = hintPath
	}

	if p.UserName == "" && p.ValidationClientName != "" {
		p.UserName = p.ValidationClientName
	}

	if p.UserKey == "" && p.ValidationKey != "" {
		p.UserKey = p.ValidationKey
	}

	if attrs, ok := c.Config["attributes_json"].(string); ok {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(attrs), &m); err != nil {
			return nil, fmt.Errorf("Error parsing attributes_json: %v", err)
		}
		p.attributes = m
	}

	if vaults, ok := c.Config["vault_json"].(string); ok {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(vaults), &m); err != nil {
			return nil, fmt.Errorf("Error parsing vault_json: %v", err)
		}

		v := make(map[string][]string)
		for vault, items := range m {
			switch items := items.(type) {
			case []interface{}:
				for _, item := range items {
					if item, ok := item.(string); ok {
						v[vault] = append(v[vault], item)
					}
				}
			case interface{}:
				if item, ok := items.(string); ok {
					v[vault] = append(v[vault], item)
				}
			}
		}

		p.vaults = v
	}

	return p, nil
}

func (p *Provisioner) deployConfigFiles(
	o terraform.UIOutput,
	comm communicator.Communicator,
	confDir string) error {
	// Copy the user key to the new instance
	pk := strings.NewReader(p.UserKey)
	if err := comm.Upload(path.Join(confDir, p.UserName+".pem"), pk); err != nil {
		return fmt.Errorf("Uploading user key failed: %v", err)
	}

	if p.SecretKey != "" {
		// Copy the secret key to the new instance
		s := strings.NewReader(p.SecretKey)
		if err := comm.Upload(path.Join(confDir, secretKey), s); err != nil {
			return fmt.Errorf("Uploading %s failed: %v", secretKey, err)
		}
	}

	// Make sure the SSLVerifyMode value is written as a symbol
	if p.SSLVerifyMode != "" && !strings.HasPrefix(p.SSLVerifyMode, ":") {
		p.SSLVerifyMode = fmt.Sprintf(":%s", p.SSLVerifyMode)
	}

	// Make strings.Join available for use within the template
	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	// Create a new template and parse the client config into it
	t := template.Must(template.New(clienrb).Funcs(funcMap).Parse(clientConf))

	var buf bytes.Buffer
	err := t.Execute(&buf, p)
	if err != nil {
		return fmt.Errorf("Error executing %s template: %s", clienrb, err)
	}

	// Copy the client config to the new instance
	if err := comm.Upload(path.Join(confDir, clienrb), &buf); err != nil {
		return fmt.Errorf("Uploading %s failed: %v", clienrb, err)
	}

	// Create a map with first boot settings
	fb := make(map[string]interface{})
	if p.attributes != nil {
		fb = p.attributes
	}

	// Check if the run_list was also in the attributes and if so log a warning
	// that it will be overwritten with the value of the run_list argument.
	if _, found := fb["run_list"]; found {
		log.Printf("[WARNING] Found a 'run_list' specified in the configured attributes! " +
			"This value will be overwritten by the value of the `run_list` argument!")
	}

	// Add the initial runlist to the first boot settings
	if !p.UsePolicyfile {
		fb["run_list"] = p.RunList
	}

	// Marshal the first boot settings to JSON
	d, err := json.Marshal(fb)
	if err != nil {
		return fmt.Errorf("Failed to create %s data: %s", firstBoot, err)
	}

	// Copy the first-boot.json to the new instance
	if err := comm.Upload(path.Join(confDir, firstBoot), bytes.NewReader(d)); err != nil {
		return fmt.Errorf("Uploading %s failed: %v", firstBoot, err)
	}

	return nil
}

func (p *Provisioner) deployOhaiHints(
	o terraform.UIOutput,
	comm communicator.Communicator,
	hintDir string) error {
	for _, hint := range p.OhaiHints {
		// Open the hint file
		f, err := os.Open(hint)
		if err != nil {
			return err
		}
		defer f.Close()

		// Copy the hint to the new instance
		if err := comm.Upload(path.Join(hintDir, path.Base(hint)), f); err != nil {
			return fmt.Errorf("Uploading %s failed: %v", path.Base(hint), err)
		}
	}

	return nil
}

func (p *Provisioner) fetchChefCertificatesFunc(
	knifeCmd string,
	confDir string) func(terraform.UIOutput, communicator.Communicator) error {
	return func(o terraform.UIOutput, comm communicator.Communicator) error {
		clientrb := path.Join(confDir, clienrb)
		cmd := fmt.Sprintf("%s ssl fetch -c %s", knifeCmd, clientrb)

		return p.runCommand(o, comm, cmd)
	}
}

func (p *Provisioner) generateClientKeyFunc(
	knifeCmd string,
	confDir string,
	noOutput string) func(terraform.UIOutput, communicator.Communicator) error {
	return func(o terraform.UIOutput, comm communicator.Communicator) error {
		options := fmt.Sprintf("-c %s -u %s --key %s",
			path.Join(confDir, clienrb),
			p.UserName,
			path.Join(confDir, p.UserName+".pem"),
		)

		// See if we already have a node object
		getNodeCmd := fmt.Sprintf("%s node show %s %s %s", knifeCmd, p.NodeName, options, noOutput)
		node := p.runCommand(o, comm, getNodeCmd) == nil

		// See if we already have a client object
		getClientCmd := fmt.Sprintf("%s client show %s %s %s", knifeCmd, p.NodeName, options, noOutput)
		client := p.runCommand(o, comm, getClientCmd) == nil

		// If we have a client, we can only continue if we are to recreate the client
		if client && !p.RecreateClient {
			return fmt.Errorf(
				"Chef client %q already exists, set recreate_client=true to automatically recreate the client", p.NodeName)
		}

		// If the node exists, try to delete it
		if node {
			deleteNodeCmd := fmt.Sprintf("%s node delete %s -y %s",
				knifeCmd,
				p.NodeName,
				options,
			)
			if err := p.runCommand(o, comm, deleteNodeCmd); err != nil {
				return err
			}
		}

		// If the client exists, try to delete it
		if client {
			deleteClientCmd := fmt.Sprintf("%s client delete %s -y %s",
				knifeCmd,
				p.NodeName,
				options,
			)
			if err := p.runCommand(o, comm, deleteClientCmd); err != nil {
				return err
			}
		}

		// Create the new client object
		createClientCmd := fmt.Sprintf("%s client create %s -d -f %s %s",
			knifeCmd,
			p.NodeName,
			path.Join(confDir, "client.pem"),
			options,
		)

		return p.runCommand(o, comm, createClientCmd)
	}
}

func (p *Provisioner) configureVaultsFunc(
	gemCmd string,
	knifeCmd string,
	confDir string) func(terraform.UIOutput, communicator.Communicator) error {
	return func(o terraform.UIOutput, comm communicator.Communicator) error {
		if err := p.runCommand(o, comm, fmt.Sprintf("%s install chef-vault", gemCmd)); err != nil {
			return err
		}

		options := fmt.Sprintf("-c %s -u %s --key %s",
			path.Join(confDir, clienrb),
			p.UserName,
			path.Join(confDir, p.UserName+".pem"),
		)

		for vault, items := range p.vaults {
			for _, item := range items {
				updateCmd := fmt.Sprintf("%s vault update %s %s -A %s -M client %s",
					knifeCmd,
					vault,
					item,
					p.NodeName,
					options,
				)
				if err := p.runCommand(o, comm, updateCmd); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func (p *Provisioner) runChefClientFunc(
	chefCmd string,
	confDir string) func(terraform.UIOutput, communicator.Communicator) error {
	return func(o terraform.UIOutput, comm communicator.Communicator) error {
		fb := path.Join(confDir, firstBoot)
		var cmd string

		// Policyfiles do not support chef environments, so don't pass the `-E` flag.
		switch {
		case p.UsePolicyfile && p.NamedRunList == "":
			cmd = fmt.Sprintf("%s -j %q", chefCmd, fb)
		case p.UsePolicyfile && p.NamedRunList != "":
			cmd = fmt.Sprintf("%s -j %q -n %q", chefCmd, fb, p.NamedRunList)
		default:
			cmd = fmt.Sprintf("%s -j %q -E %q", chefCmd, fb, p.Environment)
		}

		if p.LogToFile {
			if err := os.MkdirAll(logfileDir, 0755); err != nil {
				return fmt.Errorf("Error creating logfile directory %s: %v", logfileDir, err)
			}

			logFile := path.Join(logfileDir, p.NodeName)
			f, err := os.Create(path.Join(logFile))
			if err != nil {
				return fmt.Errorf("Error creating logfile %s: %v", logFile, err)
			}
			f.Close()

			o.Output("Writing Chef Client output to " + logFile)
			o = p
		}

		return p.runCommand(o, comm, cmd)
	}
}

// Output implementation of terraform.UIOutput interface
func (p *Provisioner) Output(output string) {
	logFile := path.Join(logfileDir, p.NodeName)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Error creating logfile %s: %v", logFile, err)
		return
	}
	defer f.Close()

	// These steps are needed to remove any ANSI escape codes used to colorize
	// the output and to make sure we have proper line endings before writing
	// the string to the logfile.
	re := regexp.MustCompile(`\x1b\[[0-9;]+m`)
	output = re.ReplaceAllString(output, "")
	output = strings.Replace(output, "\r", "\n", -1)

	if _, err := f.WriteString(output); err != nil {
		log.Printf("Error writing output to logfile %s: %v", logFile, err)
	}

	if err := f.Sync(); err != nil {
		log.Printf("Error saving logfile %s to disk: %v", logFile, err)
	}
}

// runCommand is used to run already prepared commands
func (p *Provisioner) runCommand(
	o terraform.UIOutput,
	comm communicator.Communicator,
	command string) error {
	// Unless prevented, prefix the command with sudo
	if p.useSudo {
		command = "sudo " + command
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go p.copyOutput(o, outR, outDoneCh)
	go p.copyOutput(o, errR, errDoneCh)

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

func (p *Provisioner) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

// retryFunc is used to retry a function for a given duration
func retryFunc(timeout time.Duration, f func() error) error {
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
