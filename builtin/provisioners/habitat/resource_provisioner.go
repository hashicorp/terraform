package habitat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const installURL = "https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"
const systemdUnit = `
[Unit]
Description=Habitat Supervisor

[Service]
ExecStart=/bin/hab sup run {{ .SupOptions }}
Restart=on-failure
{{ if .GatewayAuthToken -}}
Environment=HAB_SUP_GATEWAY_AUTH_TOKEN={{ .GatewayAuthToken }}
{{ end -}}
{{ if .BuilderAuthToken -}}
Environment="HAB_AUTH_TOKEN={{ .BuilderAuthToken }}"
{{ end -}}
{{ if .License -}}
Environment="HAB_LICENSE={{ .License }}"
{{ end -}}

[Install]
WantedBy=default.target
`

type provisioner struct {
	Version          string
	License          string
	AutoUpdate       bool
	HttpDisable      bool
	Services         []Service
	PermanentPeer    bool
	ListenCtl        string
	ListenGossip     string
	ListenHTTP       string
	Peers            []string
	RingKey          string
	RingKeyContent   string
	CtlSecret        string
	SkipInstall      bool
	UseSudo          bool
	ServiceType      string
	ServiceName      string
	URL              string
	Channel          string
	Events           string
	Organization     string
	GatewayAuthToken string
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
			"license": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"auto_update": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"http_disable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"peers": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"service_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "systemd",
				ValidateFunc: validation.StringInSlice([]string{"systemd", "unmanaged"}, false),
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
			"listen_ctl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
			"ctl_secret": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					_, err := url.Parse(val.(string))
					if err != nil {
						errs = append(errs, err)
					}
					return warns, errs
				},
			},
			"channel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"events": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"organization": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"gateway_auth_token": &schema.Schema{
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
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"leader", "standalone"}, false),
						},
						"user_toml": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"strategy": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"none", "rolling", "at-once"}, false),
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
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								_, err := url.Parse(val.(string))
								if err != nil {
									errs = append(errs, err)
								}
								return warns, errs
							},
						},
						"application": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"environment": &schema.Schema{
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

	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	err = communicator.Retry(retryCtx, func() error {
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

	if p.CtlSecret != "" {
		o.Output("Uploading ctl secret...")
		if err := p.uploadCtlSecret(o, comm); err != nil {
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
	//ringKey, ok := c.Get("ring_key")
	//if ok && ringKey != "" && ringKey != hcl2shim.UnknownVariableValue {
	//	ringKeyContent, ringKeyContentOk := c.Get("ring_key_content")
	//	if ringKeyContentOk && ringKeyContent == "" {
	//		es = append(es, errors.New("if ring_key is specified, ring_key_content must also be specified"))
	//	}
	//}
	//
	//ringKeyContent, ok := c.Get("ring_key_content")
	//if ok && ringKeyContent != "" && ringKeyContent != hcl2shim.UnknownVariableValue {
	//	ringKey, ringOk := c.Get("ring_key")
	//	if ringOk && ringKey == "" {
	//		es = append(es, errors.New("if ring_key_content is specified, ring_key must also be specified"))
	//	}
	//}

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
		License:          d.Get("license").(string),
		AutoUpdate:       d.Get("auto_update").(bool),
		HttpDisable:      d.Get("http_disable").(bool),
		Peers:            getPeers(d.Get("peers").([]interface{})),
		Services:         getServices(d.Get("service").(*schema.Set).List()),
		UseSudo:          d.Get("use_sudo").(bool),
		ServiceType:      d.Get("service_type").(string),
		ServiceName:      d.Get("service_name").(string),
		RingKey:          d.Get("ring_key").(string),
		RingKeyContent:   d.Get("ring_key_content").(string),
		CtlSecret:        d.Get("ctl_secret").(string),
		PermanentPeer:    d.Get("permanent_peer").(bool),
		ListenCtl:        d.Get("listen_ctl").(string),
		ListenGossip:     d.Get("listen_gossip").(string),
		ListenHTTP:       d.Get("listen_http").(string),
		URL:              d.Get("url").(string),
		Channel:          d.Get("channel").(string),
		Events:           d.Get("events").(string),
		Organization:     d.Get("organization").(string),
		BuilderAuthToken: d.Get("builder_auth_token").(string),
		GatewayAuthToken: d.Get("gateway_auth_token").(string),
	}

	return p, nil
}

func getPeers(v []interface{}) []string {
	peers := make([]string, 0, len(v))
	for _, rawPeerData := range v {
		peers = append(peers, rawPeerData.(string))
	}
	return peers
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

func (p *provisioner) uploadCtlSecret(o terraform.UIOutput, comm communicator.Communicator) error {
	destination := fmt.Sprintf("/hab/sup/default/CTL_SECRET")
	// Create the destination directory
	err := p.runCommand(o, comm, fmt.Sprintf("mkdir -p %s", filepath.Dir(destination)))
	if err != nil {
		return err
	}

	keyContent := strings.NewReader(p.CtlSecret)
	if p.UseSudo {
		tempPath := fmt.Sprintf("/tmp/CTL_SECRET")
		if err := comm.Upload(tempPath, keyContent); err != nil {
			return err
		}

		return p.runCommand(o, comm, fmt.Sprintf("mv %s %s && chown root:root %s && chmod 0600 %s", tempPath, destination, destination, destination))
	}

	return comm.Upload(destination, keyContent)
}

func (p *provisioner) uploadRingKey(o terraform.UIOutput, comm communicator.Communicator) error {
	return p.runCommand(o, comm, fmt.Sprintf(`echo -e "%s" | hab ring key import`, p.RingKeyContent))
}

func (p *provisioner) installHab(o terraform.UIOutput, comm communicator.Communicator) error {
	// Download the hab installer
	if err := p.runCommand(o, comm, fmt.Sprintf("curl --silent -L0 %s > install.sh", installURL)); err != nil {
		return err
	}

	// Run the hab install script
	var command string
	if p.Version == "" {
		command = fmt.Sprintf("bash ./install.sh ")
	} else {
		command = fmt.Sprintf("bash ./install.sh -v %s", p.Version)
	}

	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	// Create the hab user
	if err := p.createHabUser(o, comm); err != nil {
		return err
	}

	// Cleanup the installer
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

	if err := p.runCommand(o, comm, command); err != nil {
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

	if len(p.Peers) > 0 {
		options += fmt.Sprintf(" --peer %s", strings.Join(p.Peers, " --peer "))
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

	if p.HttpDisable == true {
		options += fmt.Sprintf(" --http-disable")
	}

	if p.AutoUpdate == true {
		options += fmt.Sprintf(" --auto-update")
	}

	p.SupOptions = options

	// Start hab depending on service type
	switch p.ServiceType {
	case "unmanaged":
		return p.startHabUnmanaged(o, comm, options)
	case "systemd":
		return p.startHabSystemd(o, comm, options)
	default:
		return errors.New("Unsupported service type")
	}
}

// This func is a little different than the others since we need to expose HAB_AUTH_TOKEN and HAB_LICENSE to a shell
// sub-process that's actually running the supervisor.
// @TODO: Test further
func (p *provisioner) startHabUnmanaged(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	var token string
	var license string

	// Create the sup directory for the log file
	if err := p.runCommand(o, comm, "mkdir -p /hab/sup/default && chmod o+w /hab/sup/default"); err != nil {
		return err
	}

	// Set HAB_AUTH_TOKEN if provided
	if p.BuilderAuthToken != "" {
		token = fmt.Sprintf("HAB_AUTH_TOKEN=%s", p.BuilderAuthToken)
	}

	// Set HAB_LICENSE if provided
	if p.License != "" {
		license = fmt.Sprintf("HAB_LICENSE=%s", p.License)
	}

	return p.runCommand(o, comm, fmt.Sprintf("(env %s%s setsid hab sup run %s > /hab/sup/default/sup.log 2>&1 <&1 &) ; sleep 1", token, license, options))
}

func (p *provisioner) startHabSystemd(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	// Create a new template and parse the client config into it
	unitString := template.Must(template.New("hab-supervisor.service").Parse(systemdUnit))

	var buf bytes.Buffer
	err := unitString.Execute(&buf, p)
	if err != nil {
		return fmt.Errorf("Error executing %s template: %s", "hab-supervisor.service", err)
	}

	if err := p.runCommand(o, comm, fmt.Sprintf(`echo -e "%s" | tee /etc/systemd/system/%s.service > /dev/null`, &buf, p.ServiceName)); err != nil {
		return err
	}

	return p.runCommand(o, comm, fmt.Sprintf("systemctl enable %s && systemctl start %s", p.ServiceName, p.ServiceName))
}

func (p *provisioner) createHabUser(o terraform.UIOutput, comm communicator.Communicator) error {
	var addUser bool

	// Install busybox to get us the user tools we need
	if err := p.runCommand(o, comm, fmt.Sprintf("hab install core/busybox")); err != nil {
		return err
	}

	// Check for existing hab user
	if err := p.runCommand(o, comm, fmt.Sprintf("hab pkg exec core/busybox id hab")); err != nil {
		o.Output("No existing hab user detected, creating...")
		addUser = true
	}

	if addUser {
		return p.runCommand(o, comm, fmt.Sprintf("hab pkg exec core/busybox adduser -D -g \"\" hab"))
	}

	return nil
}

// In the future we'll remove the dedicated install once the synchronous load feature in hab-sup is
// available. Until then we install here to provide output and a noisy failure mechanism because
// if you install with the pkg load, it occurs asynchronously and fails quietly.
func (p *provisioner) installHabPackage(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	var options string

	if service.Channel != "" {
		options += fmt.Sprintf(" --channel %s", service.Channel)
	}

	if service.URL != "" {
		options += fmt.Sprintf(" --url %s", service.URL)
	}

	return p.runCommand(o, comm, fmt.Sprintf("hab pkg install %s %s", service.Name, options))
}

func (p *provisioner) startHabService(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	var options string

	if err := p.installHabPackage(o, comm, service); err != nil {
		return err
	}
	if err := p.uploadUserTOML(o, comm, service); err != nil {
		return err
	}

	// Upload service group key
	if service.ServiceGroupKey != "" {
		err := p.uploadServiceGroupKey(o, comm, service.ServiceGroupKey)
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

	return p.runCommand(o, comm, fmt.Sprintf("hab svc load %s %s", service.Name, options))
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

		return p.runCommand(o, comm, fmt.Sprintf("mv %s %s", tempPath, destPath))
	}

	return comm.Upload(destPath, keyContent)
}

func (p *provisioner) uploadUserTOML(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	// Create the hab svc directory to lay down the user.toml before loading the service
	o.Output("Uploading user.toml for service: " + service.Name)
	destDir := fmt.Sprintf("/hab/user/%s/config", service.getPackageName(service.Name))
	command := fmt.Sprintf("mkdir -p %s", destDir)
	if err := p.runCommand(o, comm, command); err != nil {
		return err
	}

	userToml := strings.NewReader(service.UserTOML)

	if p.UseSudo {
		if err := comm.Upload("/tmp/user.toml", userToml); err != nil {
			return err
		}
		command = fmt.Sprintf("mv /tmp/user.toml %s", destDir)
		return p.runCommand(o, comm, command)
	}

	return comm.Upload(path.Join(destDir, "user.toml"), userToml)

}

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader) {
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	// Always set HAB_NONINTERACTIVE
	env := fmt.Sprintf("env HAB_NONINTERACTIVE=true")

	// Set license acceptance
	if p.License != "" {
		env += fmt.Sprintf(" HAB_LICENSE=%s", p.License)
	}

	// Set builder auth token
	if p.BuilderAuthToken != "" {
		env += fmt.Sprintf(" HAB_AUTH_TOKEN=%s", p.BuilderAuthToken)
	}

	if p.UseSudo {
		command = fmt.Sprintf("%s sudo -E /bin/bash -c '%s'", env, command)
	} else {
		command = fmt.Sprintf("%s /bin/bash -c '%s'", env, command)
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

	if err := comm.Start(cmd); err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
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
