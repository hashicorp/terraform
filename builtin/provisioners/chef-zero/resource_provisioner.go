package chef

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-linereader"
	"path/filepath"
)

const (
	clienrb         = "client.rb"
	defaultEnv      = "_default"
	logfileDir      = "logfiles"
	linuxChefCmd    = "chef-client"
	linuxConfDir    = "/opt/chef/0"
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

local_mode true
{{ if not .UsePolicyfile }}
cookbook_path '{{ .DefaultConfDir }}/cookbooks'
{{ end }}
node_path '{{ .DefaultConfDir }}/{{ .LocalNodesDirectory }}'
role_path '{{ .DefaultConfDir }}/roles'
data_bag_path '{{ .DefaultConfDir }}/data_bags'
rubygems_url 'http://nexus.query.consul/content/groups/rubygems'
environment_path '{{ .DefaultConfDir }}/environments'
`

type provisionFn func(terraform.UIOutput, communicator.Communicator) error

type provisioner struct {
	DNAAttributes       map[string]interface{}
	NodeAttributes      map[string]interface{}
	DirResources        string
	LocalNodesDirectory string
	InstanceId          string
	Channel             string
	ClientOptions       []string
	DisableReporting    bool
	Environment         string
	LogToFile           bool
	UsePolicyfile       bool
	PolicyGroup         string
	PolicyName          string
	HTTPProxy           string
	HTTPSProxy          string
	NamedRunList        string
	NOProxy             []string
	NodeName            string
	OhaiHints           []string
	OSType              string
	RecreateClient      bool
	PreventSudo         bool
	RunList             []string
	SecretKey           string
	SkipInstall         bool
	SkipRegister        bool
	SSLVerifyMode       string
	UserName            string
	UserKey             string
	Vaults              map[string][]string
	Version             string
	DefaultConfDir      string

	cleanupUserKeyCmd string
	createConfigFiles provisionFn
	installChefClient provisionFn

	configureVaults provisionFn
	runChefClient   provisionFn
	useSudo         bool
}

// Provisioner returns a Chef provisioner
func Provisioner() terraform.ResourceProvisioner {

	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"node_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"dir_resources": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"local_nodes_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"user_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"user_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"dna_attributes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"automatic_attributes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"default_attributes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"channel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "stable",
			},
			"client_options": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"disable_reporting": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"environment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultEnv,
			},
			"fetch_chef_certificates": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"log_to_file": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"use_policyfile": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"policy_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"policy_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"http_proxy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"https_proxy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"no_proxy": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"named_run_list": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ohai_hints": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"os_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"recreate_client": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"prevent_sudo": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"run_list": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"secret_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"skip_install": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"skip_register": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"ssl_verify_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"vault_json": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		ApplyFunc:    applyFn,
		ValidateFunc: validateFn,
	}
}

func applyFn(ctx context.Context) error {
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
	s := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	// Decode the provisioner config
	p, err := decodeConfig(d)
	if err != nil {
		return err
	}

	if p.OSType == "" {
		switch t := s.Ephemeral.ConnInfo["type"]; t {
		case "ssh", "": // The default connection type is ssh, so if the type is empty assume ssh
			p.OSType = "linux"
		case "winrm":
			p.OSType = "windows"
		default:
			return fmt.Errorf("Unsupported connection type: %s", t)
		}
	}

	// Set some values based on the targeted OS
	switch p.OSType {
	case "linux":
		p.cleanupUserKeyCmd = fmt.Sprintf("rm -f %s", path.Join(linuxConfDir, p.UserName+".pem"))
		p.createConfigFiles = p.linuxCreateConfigFiles
		p.installChefClient = p.linuxInstallChefClient
		p.DefaultConfDir = linuxConfDir
		p.configureVaults = p.configureVaultsFunc(linuxGemCmd, linuxKnifeCmd, linuxConfDir)
		p.runChefClient = p.runChefClientFunc(linuxChefCmd, linuxConfDir)
		p.useSudo = !p.PreventSudo && s.Ephemeral.ConnInfo["user"] != "root"
	case "windows":
		p.cleanupUserKeyCmd = fmt.Sprintf("cd %s && del /F /Q %s", windowsConfDir, p.UserName+".pem")
		p.createConfigFiles = p.windowsCreateConfigFiles
		p.installChefClient = p.windowsInstallChefClient
		p.DefaultConfDir = windowsConfDir
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

	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err = communicator.Retry(retryCtx, func() error {
		return comm.Connect(o)
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

	// file configured here
	o.Output("Creating configuration files...")
	if err := p.createConfigFiles(o, comm); err != nil {
		return err
	}

	if p.Vaults != nil {
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

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
	usePolicyFile := false
	if usePolicyFileRaw, ok := c.Get("use_policyfile"); ok {
		switch usePolicyFileRaw := usePolicyFileRaw.(type) {
		case bool:
			usePolicyFile = usePolicyFileRaw
		case string:
			usePolicyFileBool, err := strconv.ParseBool(usePolicyFileRaw)
			if err != nil {
				return ws, append(es, errors.New("\"use_policyfile\" must be a boolean"))
			}
			usePolicyFile = usePolicyFileBool
		default:
			return ws, append(es, errors.New("\"use_policyfile\" must be a boolean"))
		}
	}

	if !usePolicyFile && !c.IsSet("run_list") {
		es = append(es, errors.New("\"run_list\": required field is not set"))
	}
	if usePolicyFile && !c.IsSet("policy_name") {
		es = append(es, errors.New("using policyfile, but \"policy_name\" not set"))
	}
	if usePolicyFile && !c.IsSet("policy_group") {
		es = append(es, errors.New("using policyfile, but \"policy_group\" not set"))
	}

	return ws, es
}

// called by @createConfigFiles
func (p *provisioner) prepareConfigFiles(o terraform.UIOutput, comm communicator.Communicator, confDir string) error {

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
	if err = comm.Upload(path.Join(confDir, clienrb), &buf); err != nil {
		return fmt.Errorf("Uploading %s failed: %v", clienrb, err)
	}

	// Create a map with first boot settings
	dna := make(map[string]interface{})
	if p.DNAAttributes != nil {
		dna = p.DNAAttributes
	}

	node := make(map[string]interface{})
	if p.NodeAttributes != nil {
		node = p.NodeAttributes
	}

	// Check if the run_list was also in the attributes and if so log a warning
	// that it will be overwritten with the value of the run_list argument.
	if _, found := dna["run_list"]; found {
		log.Printf("[WARN] Found a 'run_list' specified in the configured attributes! " +
			"This value will be overwritten by the value of the `run_list` argument!")
	}

	// Add the initial runlist to the first boot settings
	if !p.UsePolicyfile {
		dna["run_list"] = p.RunList
		node["run_list"] = p.RunList
	} else {
		dna["policy_name"] = p.PolicyName
		dna["policy_group"] = p.PolicyGroup
		node["policy_name"] = p.PolicyName
		node["policy_group"] = p.PolicyGroup
	}

	createChefJsonFile(o, p.DirResources+"/dna", dna, p.NodeAttributes["id"].(string))
	createChefJsonFile(o, p.DirResources+"/"+p.LocalNodesDirectory, node, p.NodeAttributes["id"].(string))

	return nil
}

func createChefJsonFile(o terraform.UIOutput, dir string, b map[string]interface{}, filename string) error {
	nodefile := filename + ".json"
	var d, err = json.Marshal(b)
	if err != nil {
		return fmt.Errorf("Failed to create %s data: %s", nodefile, err)
	}

	nodePath, err := homedir.Expand(path.Join(dir, nodefile))
	_, err = os.Stat(nodePath)
	if err == nil {
		return fmt.Errorf("File %s already exist", nodePath)
	}

	f, err := os.Create(path.Join(nodePath))

	if err != nil {
		return fmt.Errorf("Error creating file %s: %v", nodePath, err)
	}

	o.Output("Writing " + nodePath)
	_, err = f.Write(d)
	o.Output("File written " + nodePath)

	if err != nil {
		return fmt.Errorf("Failed to write data %d to node file %s: %v", d, nodePath, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("Error closing node file %s: %v", nodePath, err)
	}
	return nil
}

func mapDynamicVariables(node map[string]interface{}, dynamic map[string]interface{}) map[string]interface{} {
	for ktm, vtm := range node {
		if f, ok := vtm.(string); ok {
			node[ktm] = mapDynamicStringValue(f, dynamic)
		} else if f, ok := vtm.([]string); ok {
			for i, s := range f {
				f[i] = mapDynamicStringValue(s, dynamic)
			}
			node[ktm] = f
		} else if f, ok := vtm.([]map[string]interface{}); ok {
			for i, arr := range f {
				f[i] = mapDynamicVariables(arr, dynamic)
			}
			node[ktm] = f
		} else if f, ok := vtm.(map[string]interface{}); ok {
			node[ktm] = mapDynamicVariables(f, dynamic)
		}
	}
	return node
}

func mapDynamicStringValue(s string, dynamic map[string]interface{}) string {
	for k, v := range dynamic {
		if s == "<%= @"+k+" %>" {
			if va, ok := v.(string); ok {
				return va
			}
		}
	}
	return s
}

func (p *provisioner) deployDirectoryFiles(o terraform.UIOutput, comm communicator.Communicator, confDir, dirPath string) error {

	_, err := os.Stat(path.Join(p.DirResources, dirPath))

	if os.IsNotExist(err) || err != nil {
		o.Output("Warning: " + path.Join(p.DirResources, dirPath) + " does not exist, uploading nothing.")
		return nil
	}
	var files, _ = filepath.Glob(path.Join(p.DirResources, dirPath))
	o.Output("Uploading " + path.Join(p.DirResources, dirPath) + " to " + confDir + "\n" +
		"file list : " + strings.Join(files, ","))

	if err := comm.UploadDir(confDir, path.Join(p.DirResources, dirPath)); err != nil {
		return fmt.Errorf("Uploading %s failed: %v", path.Base(dirPath), err)
	}

	return nil
}

func (p *provisioner) deployOhaiHints(o terraform.UIOutput, comm communicator.Communicator, hintDir string) error {
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

func (p *provisioner) configureVaultsFunc(gemCmd string, knifeCmd string, confDir string) provisionFn {
	return func(o terraform.UIOutput, comm communicator.Communicator) error {
		if err := p.runCommand(o, comm, fmt.Sprintf("%s install chef-vault", gemCmd)); err != nil {
			return err
		}

		options := fmt.Sprintf("-c %s -u %s --key %s",
			path.Join(confDir, clienrb),
			p.UserName,
			path.Join(confDir, p.UserName+".pem"),
		)

		// if client gets recreated, remove (old) client (with old keys) from vaults/items
		// otherwise, the (new) client (with new keys) will not be able to decrypt the vault
		if p.RecreateClient {
			for vault, items := range p.Vaults {
				for _, item := range items {
					deleteCmd := fmt.Sprintf("%s vault remove %s %s -C \"%s\" -M client %s",
						knifeCmd,
						vault,
						item,
						p.NodeName,
						options,
					)
					if err := p.runCommand(o, comm, deleteCmd); err != nil {
						return err
					}
				}
			}
		}

		for vault, items := range p.Vaults {
			for _, item := range items {
				updateCmd := fmt.Sprintf("%s vault update %s %s -C %s -M client %s",
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

func (p *provisioner) runChefClientFunc(chefCmd string, confDir string) provisionFn {
	return func(o terraform.UIOutput, comm communicator.Communicator) error {
		fb := path.Join(confDir+"/dna/", p.NodeAttributes["id"].(string)+".json")
		var cmd string
		var pipelog string = ""

		// Policyfiles do not support chef environments, so don't pass the `-E` flag.
		switch {

		case p.UsePolicyfile && p.NamedRunList == "":
			cmd = fmt.Sprintf("%s -z -c %s -j %q", chefCmd, path.Join(confDir, clienrb), fb)
		case p.UsePolicyfile && p.NamedRunList != "":
			cmd = fmt.Sprintf("%s -z -c %s -j %q -n %q", chefCmd, path.Join(confDir, clienrb), fb, pipelog)
		default:
			cmd = fmt.Sprintf("%s -z -c %s -j %q -E %q", chefCmd, path.Join(confDir, clienrb), fb, p.Environment)
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

		return p.runCommand(o, comm, fmt.Sprintf("cd %s/cookbooks && %s", confDir, cmd))
	}
}

// Output implementation of terraform.UIOutput interface
func (p *provisioner) Output(output string) {
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
func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	// Unless prevented, prefix the command with sudo
	if p.useSudo {
		command = "sudo bash -c '" + command + "'"
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	go p.copyOutput(o, outR)
	go p.copyOutput(o, errR)
	defer outW.Close()
	defer errW.Close()

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err := comm.Start(cmd)
	if err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader) {
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		Channel:             d.Get("channel").(string),
		ClientOptions:       getStringList(d.Get("client_options")),
		DisableReporting:    d.Get("disable_reporting").(bool),
		Environment:         d.Get("environment").(string),
		LogToFile:           d.Get("log_to_file").(bool),
		UsePolicyfile:       d.Get("use_policyfile").(bool),
		PolicyGroup:         d.Get("policy_group").(string),
		PolicyName:          d.Get("policy_name").(string),
		HTTPProxy:           d.Get("http_proxy").(string),
		HTTPSProxy:          d.Get("https_proxy").(string),
		NOProxy:             getStringList(d.Get("no_proxy")),
		NamedRunList:        d.Get("named_run_list").(string),
		NodeName:            d.Get("node_name").(string),
		OhaiHints:           getStringList(d.Get("ohai_hints")),
		OSType:              d.Get("os_type").(string),
		RecreateClient:      d.Get("recreate_client").(bool),
		PreventSudo:         d.Get("prevent_sudo").(bool),
		RunList:             getStringList(d.Get("run_list")),
		SecretKey:           d.Get("secret_key").(string),
		SkipInstall:         d.Get("skip_install").(bool),
		SkipRegister:        d.Get("skip_register").(bool),
		SSLVerifyMode:       d.Get("ssl_verify_mode").(string),
		UserName:            d.Get("user_name").(string),
		UserKey:             d.Get("user_key").(string),
		Version:             d.Get("version").(string),
		InstanceId:          d.Get("instance_id").(string),
		LocalNodesDirectory: d.Get("local_nodes_dir").(string),
		DirResources:        d.Get("dir_resources").(string),
	}

	for i, hint := range p.OhaiHints {
		hintPath, err := homedir.Expand(hint)
		if err != nil {
			return nil, fmt.Errorf("Error expanding the path %s: %v", hint, err)
		}
		p.OhaiHints[i] = hintPath
	}

	p.NodeAttributes = make(map[string]interface{})

	for _, dir := range []string{"data_bags", p.LocalNodesDirectory, "roles", "dna", "environments", "cookbooks"} {
		configPath, err := homedir.Expand(dir)

		if err != nil {
			return nil, fmt.Errorf("Error expanding the path %s: %v", configPath, err)
		}

	}

	if _, err := d.GetOk("instance_id"); err != true {
		return nil, fmt.Errorf("Error parsing instance_id")
	} else {
		p.NodeAttributes["id"] = d.Get("instance_id").(string)
	}

	types := []string{"automatic", "default", "force_default", "normal", "override", "force_override"}
	for _, v := range types {
		if attrs, ok := d.GetOk(v + "_attributes"); ok {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(attrs.(string)), &m); err != nil {
				return nil, fmt.Errorf("Error parsing %s_attributes: %v", v, err)
			}
			p.NodeAttributes[v] = m
		}
	}

	if attrs, ok := d.GetOk("dna_attributes"); ok {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(attrs.(string)), &m); err != nil {
			return nil, fmt.Errorf("Error parsing attributes_dna: %v", err)
		}
		p.DNAAttributes = m
	}

	// Check if nodes directory doesn't already exist
	if _, ok := d.GetOk("local_nodes_dir"); ok {
		path, err := homedir.Expand(p.DirResources + "/" + p.LocalNodesDirectory)

		if err != nil {
			return nil, fmt.Errorf("Error expanding node directory %s: %v", path, err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if errmk := os.Mkdir(path, 0755); errmk != nil {
				return nil, fmt.Errorf("Error creating node directory %s: %v", path, err)
			}
		}

		path, err = homedir.Expand(p.DirResources + "/dna")

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if errmk := os.Mkdir(path, 0755); errmk != nil {
				return nil, fmt.Errorf("Error creating dna directory %s: %v", path, err)
			}
		}
	}

	if vaults, ok := d.GetOk("vault_json"); ok {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(vaults.(string)), &m); err != nil {
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

		p.Vaults = v
	}

	return p, nil
}

func getStringList(v interface{}) []string {
	var result []string

	switch v := v.(type) {
	case nil:
		return result
	case []interface{}:
		for _, vv := range v {
			if vv, ok := vv.(string); ok {
				result = append(result, vv)
			}
		}
		return result
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}
}
