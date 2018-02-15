package habitat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	linereader "github.com/mitchellh/go-linereader"
)

const installURL = "https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"
const systemdUnit = `
[Unit]
Description=Habitat Supervisor

[Service]
ExecStart=/bin/hab sup run {{ .SupOptions }}
Restart=on-failure
{{ if .BuilderAuthToken -}}
Environment="HAB_AUTH_TOKEN={{ .BuilderAuthToken }}"
{{ end -}}

[Install]
WantedBy=default.target
`

var serviceTypes = map[string]bool{"unmanaged": true, "systemd": true}
var updateStrategies = map[string]bool{"at-once": true, "rolling": true, "none": true}
var topologies = map[string]bool{"leader": true, "standalone": true}

type provisionFn func(terraform.UIOutput, communicator.Communicator) error

type provisioner struct {
	Version          string
	Services         []Service
	PermanentPeer    bool
	ListenGossip     string
	ListenHTTP       string
	Peer             string
	RingKey          string
	RingKeyContent   string
	SkipInstall      bool
	UseSudo          bool
	ServiceType      string
	ServiceName      string
	URL              string
	Channel          string
	Events           string
	OverrideName     string
	Organization     string
	BuilderAuthToken string
	SupOptions       string
}

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"peer": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"service_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "systemd",
			},
			"service_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "hab-supervisor",
			},
			"use_sudo": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"permanent_peer": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"listen_gossip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"listen_http": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ring_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ring_key_content": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"channel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"events": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"override_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"organization": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"builder_auth_token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"service": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"binds": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"bind": &schema.Schema{
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"alias": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"service": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"group": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Optional: true,
						},
						"topology": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_toml": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"strategy": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"channel": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"url": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"application": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"environment": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"override_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"service_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
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

	p, err := decodeConfig(d)
	if err != nil {
		return err
	}

	comm, err := communicator.New(s)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	err = communicator.Retry(ctx, func() error {
		return comm.Connect(o)
	})

	if err != nil {
		return err
	}
	defer comm.Disconnect()

	if !p.SkipInstall {
		o.Output("Installing habitat...")
		if err := p.installHab(o, comm); err != nil {
			return err
		}
	}

	if p.RingKeyContent != "" {
		o.Output("Uploading supervisor ring key...")
		if err := p.uploadRingKey(o, comm); err != nil {
			return err
		}
	}

	o.Output("Starting the habitat supervisor...")
	if err := p.startHab(o, comm); err != nil {
		return err
	}

	if p.Services != nil {
		for _, service := range p.Services {
			o.Output("Starting service: " + service.Name)
			if err := p.startHabService(o, comm, service); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
	serviceType, ok := c.Get("service_type")
	if ok {
		if !serviceTypes[serviceType.(string)] {
			es = append(es, errors.New(serviceType.(string)+" is not a valid service_type."))
		}
	}

	builderURL, ok := c.Get("url")
	if ok {
		if _, err := url.ParseRequestURI(builderURL.(string)); err != nil {
			es = append(es, errors.New(builderURL.(string)+" is not a valid URL."))
		}
	}

	// Validate service level configs
	services, ok := c.Get("service")
	if ok {
		for _, service := range services.([]map[string]interface{}) {
			strategy, ok := service["strategy"].(string)
			if ok && !updateStrategies[strategy] {
				es = append(es, errors.New(strategy+" is not a valid update strategy."))
			}

			topology, ok := service["topology"].(string)
			if ok && !topologies[topology] {
				es = append(es, errors.New(topology+" is not a valid topology"))
			}

			builderURL, ok := service["url"].(string)
			if ok {
				if _, err := url.ParseRequestURI(builderURL); err != nil {
					es = append(es, errors.New(builderURL+" is not a valid URL."))
				}
			}
		}
	}
	return ws, es
}

type Service struct {
	Name            string
	Strategy        string
	Topology        string
	Channel         string
	Group           string
	URL             string
	Binds           []Bind
	BindStrings     []string
	UserTOML        string
	AppName         string
	Environment     string
	OverrideName    string
	ServiceGroupKey string
}

type Bind struct {
	Alias   string
	Service string
	Group   string
}

func (s *Service) getPackageName(fullName string) string {
	return strings.Split(fullName, "/")[1]
}

func (b *Bind) toBindString() string {
	return fmt.Sprintf("%s:%s.%s", b.Alias, b.Service, b.Group)
}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		Version:          d.Get("version").(string),
		Peer:             d.Get("peer").(string),
		Services:         getServices(d.Get("service").(*schema.Set).List()),
		UseSudo:          d.Get("use_sudo").(bool),
		ServiceType:      d.Get("service_type").(string),
		ServiceName:      d.Get("service_name").(string),
		RingKey:          d.Get("ring_key").(string),
		RingKeyContent:   d.Get("ring_key_content").(string),
		PermanentPeer:    d.Get("permanent_peer").(bool),
		ListenGossip:     d.Get("listen_gossip").(string),
		ListenHTTP:       d.Get("listen_http").(string),
		URL:              d.Get("url").(string),
		Channel:          d.Get("channel").(string),
		Events:           d.Get("events").(string),
		OverrideName:     d.Get("override_name").(string),
		Organization:     d.Get("organization").(string),
		BuilderAuthToken: d.Get("builder_auth_token").(string),
	}

	return p, nil
}

func getServices(v []interface{}) []Service {
	services := make([]Service, 0, len(v))
	for _, rawServiceData := range v {
		serviceData := rawServiceData.(map[string]interface{})
		name := (serviceData["name"].(string))
		strategy := (serviceData["strategy"].(string))
		topology := (serviceData["topology"].(string))
		channel := (serviceData["channel"].(string))
		group := (serviceData["group"].(string))
		url := (serviceData["url"].(string))
		app := (serviceData["application"].(string))
		env := (serviceData["environment"].(string))
		override := (serviceData["override_name"].(string))
		userToml := (serviceData["user_toml"].(string))
		serviceGroupKey := (serviceData["service_key"].(string))
		var bindStrings []string
		binds := getBinds(serviceData["bind"].(*schema.Set).List())
		for _, b := range serviceData["binds"].([]interface{}) {
			bind, err := getBindFromString(b.(string))
			if err != nil {
				return nil
			}
			binds = append(binds, bind)
		}

		service := Service{
			Name:            name,
			Strategy:        strategy,
			Topology:        topology,
			Channel:         channel,
			Group:           group,
			URL:             url,
			UserTOML:        userToml,
			BindStrings:     bindStrings,
			Binds:           binds,
			AppName:         app,
			Environment:     env,
			OverrideName:    override,
			ServiceGroupKey: serviceGroupKey,
		}
		services = append(services, service)
	}
	return services
}

func getBinds(v []interface{}) []Bind {
	binds := make([]Bind, 0, len(v))
	for _, rawBindData := range v {
		bindData := rawBindData.(map[string]interface{})
		alias := bindData["alias"].(string)
		service := bindData["service"].(string)
		group := bindData["group"].(string)
		bind := Bind{
			Alias:   alias,
			Service: service,
			Group:   group,
		}
		binds = append(binds, bind)
	}
	return binds
}

func (p *provisioner) uploadRingKey(o terraform.UIOutput, comm communicator.Communicator) error {
	command := fmt.Sprintf("echo '%s' | hab ring key import", p.RingKeyContent)
	if p.UseSudo {
		command = fmt.Sprintf("echo '%s' | sudo hab ring key import", p.RingKeyContent)
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) installHab(o terraform.UIOutput, comm communicator.Communicator) error {
	// Build the install command
	command := fmt.Sprintf("curl -L0 %s > install.sh", installURL)
	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	// Run the install script
	if p.Version == "" {
		command = fmt.Sprintf("env HAB_NONINTERACTIVE=true bash ./install.sh ")
	} else {
		command = fmt.Sprintf("env HAB_NONINTERACTIVE=true bash ./install.sh -v %s", p.Version)
	}

	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	if err := p.createHabUser(o, comm); err != nil {
		return err
	}

	return p.runCommand(o, comm, fmt.Sprintf("rm -f install.sh"))
}

func (p *provisioner) startHab(o terraform.UIOutput, comm communicator.Communicator) error {
	// Install the supervisor first
	var command string
	if p.Version == "" {
		command += fmt.Sprintf("hab install core/hab-sup")
	} else {
		command += fmt.Sprintf("hab install core/hab-sup/%s", p.Version)
	}

	if p.UseSudo {
		command = fmt.Sprintf("sudo -E %s", command)
	}

	command = fmt.Sprintf("env HAB_NONINTERACTIVE=true %s", command)

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	// Build up sup options
	options := ""
	if p.PermanentPeer {
		options += " -I"
	}

	if p.ListenGossip != "" {
		options += fmt.Sprintf(" --listen-gossip %s", p.ListenGossip)
	}

	if p.ListenHTTP != "" {
		options += fmt.Sprintf(" --listen-http %s", p.ListenHTTP)
	}

	if p.Peer != "" {
		options += fmt.Sprintf(" --peer %s", p.Peer)
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

	if p.OverrideName != "" {
		options += fmt.Sprintf(" --override-name %s", p.OverrideName)
	}

	if p.Organization != "" {
		options += fmt.Sprintf(" --org %s", p.Organization)
	}

	p.SupOptions = options

	switch p.ServiceType {
	case "unmanaged":
		return p.startHabUnmanaged(o, comm, options)
	case "systemd":
		return p.startHabSystemd(o, comm, options)
	default:
		return errors.New("Unsupported service type")
	}
}

func (p *provisioner) startHabUnmanaged(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	// Create the sup directory for the log file
	var command string
	var token string
	if p.UseSudo {
		command = "sudo mkdir -p /hab/sup/default && sudo chmod o+w /hab/sup/default"
	} else {
		command = "mkdir -p /hab/sup/default && chmod o+w /hab/sup/default"
	}
	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	if p.BuilderAuthToken != "" {
		token = fmt.Sprintf("env HAB_AUTH_TOKEN=%s", p.BuilderAuthToken)
	}

	if p.UseSudo {
		command = fmt.Sprintf("(%s setsid sudo -E hab sup run %s > /hab/sup/default/sup.log 2>&1 &) ; sleep 1", token, options)
	} else {
		command = fmt.Sprintf("(%s setsid hab sup run %s > /hab/sup/default/sup.log 2>&1 <&1 &) ; sleep 1", token, options)
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) startHabSystemd(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	// Create a new template and parse the client config into it
	unitString := template.Must(template.New("hab-supervisor.service").Parse(systemdUnit))

	var buf bytes.Buffer
	err := unitString.Execute(&buf, p)
	if err != nil {
		return fmt.Errorf("Error executing %s template: %s", "hab-supervisor.service", err)
	}

	var command string
	if p.UseSudo {
		command = fmt.Sprintf("sudo echo '%s' | sudo tee /etc/systemd/system/%s.service > /dev/null", &buf, p.ServiceName)
	} else {
		command = fmt.Sprintf("echo '%s' | tee /etc/systemd/system/%s.service > /dev/null", &buf, p.ServiceName)
	}

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	if p.UseSudo {
		command = fmt.Sprintf("sudo systemctl start %s", p.ServiceName)
	} else {
		command = fmt.Sprintf("systemctl start %s", p.ServiceName)
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) createHabUser(o terraform.UIOutput, comm communicator.Communicator) error {
	addUser := false
	// Install busybox to get us the user tools we need
	command := fmt.Sprintf("env HAB_NONINTERACTIVE=true hab install core/busybox")
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	// Check for existing hab user
	command = fmt.Sprintf("hab pkg exec core/busybox id hab")
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	if err := p.runCommand(o, comm, command); err != nil {
		o.Output("No existing hab user detected, creating...")
		addUser = true
	}

	if addUser {
		command = fmt.Sprintf("hab pkg exec core/busybox adduser -D -g \"\" hab")
		if p.UseSudo {
			command = fmt.Sprintf("sudo %s", command)
		}
		return p.runCommand(o, comm, command)
	}

	return nil
}

func (p *provisioner) startHabService(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	var command string
	if p.UseSudo {
		command = fmt.Sprintf("env HAB_NONINTERACTIVE=true sudo -E hab pkg install %s", service.Name)
	} else {
		command = fmt.Sprintf("env HAB_NONINTERACTIVE=true hab pkg install %s", service.Name)
	}

	if p.BuilderAuthToken != "" {
		command = fmt.Sprintf("env HAB_AUTH_TOKEN=%s %s", p.BuilderAuthToken, command)
	}

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	if err := p.uploadUserTOML(o, comm, service); err != nil {
		return err
	}

	// Upload service group key
	if service.ServiceGroupKey != "" {
		p.uploadServiceGroupKey(o, comm, service.ServiceGroupKey)
	}

	options := ""
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
		options += fmt.Sprintf("--url %s", service.URL)
	}

	if service.Group != "" {
		options += fmt.Sprintf(" --group %s", service.Group)
	}

	for _, bind := range service.Binds {
		options += fmt.Sprintf(" --bind %s", bind.toBindString())
	}
	command = fmt.Sprintf("hab svc load %s %s", service.Name, options)
	if p.UseSudo {
		command = fmt.Sprintf("sudo -E %s", command)
	}
	if p.BuilderAuthToken != "" {
		command = fmt.Sprintf("env HAB_AUTH_TOKEN=%s %s", p.BuilderAuthToken, command)
	}
	return p.runCommand(o, comm, command)
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
		command := fmt.Sprintf("sudo mv %s %s", tempPath, destPath)
		return p.runCommand(o, comm, command)
	}

	return comm.Upload(destPath, keyContent)
}

func (p *provisioner) uploadUserTOML(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	// Create the hab svc directory to lay down the user.toml before loading the service
	o.Output("Uploading user.toml for service: " + service.Name)
	destDir := fmt.Sprintf("/hab/svc/%s", service.getPackageName(service.Name))
	command := fmt.Sprintf("mkdir -p %s", destDir)
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	userToml := strings.NewReader(service.UserTOML)

	if p.UseSudo {
		if err := comm.Upload("/tmp/user.toml", userToml); err != nil {
			return err
		}
		command = fmt.Sprintf("sudo mv /tmp/user.toml %s", destDir)
		return p.runCommand(o, comm, command)
	}

	return comm.Upload(path.Join(destDir, "user.toml"), userToml)

}

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	var err error

	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go p.copyOutput(o, outR, outDoneCh)
	go p.copyOutput(o, errR, errDoneCh)

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	if err = comm.Start(cmd); err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	cmd.Wait()
	if cmd.ExitStatus != 0 {
		err = fmt.Errorf(
			"Command %q exited with non-zero exit status: %d", cmd.Command, cmd.ExitStatus)
	}

	outW.Close()
	errW.Close()
	<-outDoneCh
	<-errDoneCh

	return err
}

func getBindFromString(bind string) (Bind, error) {
	t := strings.FieldsFunc(bind, func(d rune) bool {
		switch d {
		case ':', '.':
			return true
		}
		return false
	})
	if len(t) != 3 {
		return Bind{}, errors.New("Invalid bind specification: " + bind)
	}
	return Bind{Alias: t[0], Service: t[1], Group: t[2]}, nil
}
