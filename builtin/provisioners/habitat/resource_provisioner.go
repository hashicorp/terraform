package habitat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	linereader "github.com/mitchellh/go-linereader"
)

const install_url = "https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"

var serviceTypes = map[string]bool{"unmanaged": true, "systemd": true}
var updateStrategies = map[string]bool{"at-once": true, "rolling": true, "none": true}
var topologies = map[string]bool{"leader": true, "standalone": true}

type provisionFn func(terraform.UIOutput, communicator.Communicator) error

type provisioner struct {
	Version       string    `mapstructure:"version"`
	Services      []Service `mapstructure:"service"`
	PermanentPeer bool      `mapstructure:"permanent_peer"`
	ListenGossip  string    `mapstructure:"listen_gossip"`
	ListenHTTP    string    `mapstructure:"listen_http"`
	Peer          string    `mapstructure:"peer"`
	RingKey       string    `mapstructure:"ring_key"`
	SkipInstall   bool      `mapstructure:"skip_hab_install"`
	UseSudo       bool      `mapstructure:"use_sudo"`
	ServiceType   string    `mapstructure:"service_type"`
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
				Default:  "unmanaged",
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

	p, err := decodeConfig(d, o)
	if err != nil {
		return err
	}

	comm, err := communicator.New(s)
	if err != nil {
		return err
	}

	err = retryFunc(comm.Timeout(), func() error {
		err = comm.Connect(o)
		return err
	})
	if err != nil {
		return err
	}
	defer comm.Disconnect()

	if !p.SkipInstall {
		if err := p.installHab(o, comm); err != nil {
			return err
		}
	}

	if err := p.startHab(o, comm); err != nil {
		return err
	}

	if p.Services != nil {
		for _, service := range p.Services {
			if err := p.startHabService(o, comm, service); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
	return nil, nil
}

type Service struct {
	Name        string   `mapstructure:"name"`
	Strategy    string   `mapstructure:"strategy"`
	Topology    string   `mapstructure:"topology"`
	Channel     string   `mapstructure:"channel"`
	Group       string   `mapstructure:"group"`
	URL         string   `mapstructure:"url"`
	Binds       []Bind   `mapstructure:"bind"`
	BindStrings []string `mapstructure:"binds"`
	UserTOML    string   `mapstructure:"user_toml"`
}

type Bind struct {
	Alias   string `mapstructure:"alias"`
	Service string `mapstructure:"service"`
	Group   string `mapstructure:"group"`
}

func (s *Service) getPackageName(full_name string) string {
	return strings.Split(full_name, "/")[1]
}

func (b *Bind) toBindString() string {
	return fmt.Sprintf("%s:%s.%s", b.Alias, b.Service, b.Group)
}

func decodeConfig(d *schema.ResourceData, o terraform.UIOutput) (*provisioner, error) {
	// o.Output(fmt.Sprintf("%+v\n", d))
	// o.Output(fmt.Sprintf("%+v\n", d.Get("service").(*schema.Set).List()))
	p := &provisioner{
		Version:       d.Get("version").(string),
		Peer:          d.Get("peer").(string),
		Services:      getServices(d.Get("service").(*schema.Set).List(), o),
		UseSudo:       d.Get("use_sudo").(bool),
		ServiceType:   d.Get("service_type").(string),
		RingKey:       d.Get("ring_key").(string),
		PermanentPeer: d.Get("permanent_peer").(bool),
		ListenGossip:  d.Get("listen_gossip").(string),
		ListenHTTP:    d.Get("listen_http").(string),
	}
	debug, _ := json.Marshal(p)
	o.Output(string(debug))

	return p, nil
}

func getServices(v []interface{}, o terraform.UIOutput) []Service {
	services := make([]Service, 0, len(v))
	for _, rawServiceData := range v {
		serviceData := rawServiceData.(map[string]interface{})
		name := (serviceData["name"].(string))
		strategy := (serviceData["strategy"].(string))
		topology := (serviceData["topology"].(string))
		channel := (serviceData["channel"].(string))
		group := (serviceData["group"].(string))
		url := (serviceData["url"].(string))
		userToml := (serviceData["user_toml"].(string))
		var bindStrings []string
		binds := getBinds(serviceData["bind"].(*schema.Set).List(), o)
		for _, b := range serviceData["binds"].([]interface{}) {
			// bindStrings = append(bindStrings, b.(string))
			bind, err := getBindFromString(b.(string))
			if err != nil {
				return nil
			}
			binds = append(binds, bind)
		}

		service := Service{
			Name:        name,
			Strategy:    strategy,
			Topology:    topology,
			Channel:     channel,
			Group:       group,
			URL:         url,
			UserTOML:    userToml,
			BindStrings: bindStrings,
			Binds:       binds,
		}
		services = append(services, service)
	}
	return services
}

func getBinds(v []interface{}, o terraform.UIOutput) []Bind {
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

// TODO:  Add proxy support
func (p *provisioner) installHab(o terraform.UIOutput, comm communicator.Communicator) error {
	// Build the install command
	command := fmt.Sprintf("curl -L0 %s > install.sh", install_url)
	err := p.runCommand(o, comm, command)
	if err != nil {
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

	err = p.runCommand(o, comm, command)
	if err != nil {
		return err
	}

	err = p.createHabUser(o, comm)
	if err != nil {
		return err
	}

	return p.runCommand(o, comm, fmt.Sprintf("rm -f install.sh"))
}

// TODO: Add support for options
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

	err := p.runCommand(o, comm, command)
	if err != nil {
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

	switch p.ServiceType {
	case "unmanaged":
		return p.startHabUnmanaged(o, comm, options)
	case "systemd":
		return p.startHabSystemd(o, comm, options)
	default:
		return err
	}
}

func (p *provisioner) startHabUnmanaged(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	// Create the sup directory for the log file
	var command string
	if p.UseSudo {
		command = "sudo mkdir -p /hab/sup/default && sudo chmod o+w /hab/sup/default"
	} else {
		command = "mkdir -p /hab/sup/default && chmod o+w /hab/sup/default"
	}
	err := p.runCommand(o, comm, command)
	if err != nil {
		return err
	}

	if p.UseSudo {
		command = fmt.Sprintf("(setsid sudo hab sup run %s > /hab/sup/default/sup.log 2>&1 &) ; sleep 1", options)
	} else {
		command = fmt.Sprintf("(setsid hab sup run %s > /hab/sup/default/sup.log 2>&1 <&1 &) ; sleep 1", options)
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) startHabSystemd(o terraform.UIOutput, comm communicator.Communicator, options string) error {
	systemd_unit := `[Unit]
Description=Habitat Supervisor

[Service]
ExecStart=/bin/hab sup run %s
Restart=on-failure

[Install]
WantedBy=default.target`

	systemd_unit = fmt.Sprintf(systemd_unit, options)
	var command string
	if p.UseSudo {
		command = fmt.Sprintf("sudo echo '%s' | sudo tee /etc/systemd/system/hab-supervisor.service > /dev/null", systemd_unit)
	} else {
		command = fmt.Sprintf("echo '%s' | tee /etc/systemd/system/hab-supervisor.service > /dev/null", systemd_unit)
	}

	err := p.runCommand(o, comm, command)
	if err != nil {
		return err
	}

	if p.UseSudo {
		command = fmt.Sprintf("sudo systemctl start hab-supervisor")
	} else {
		command = fmt.Sprintf("systemctl start hab-supervisor")
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) createHabUser(o terraform.UIOutput, comm communicator.Communicator) error {
	// Create the hab user
	command := fmt.Sprintf("env HAB_NONINTERACTIVE=true hab install core/busybox")
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	err := p.runCommand(o, comm, command)
	if err != nil {
		return err
	}

	command = fmt.Sprintf("hab pkg exec core/busybox adduser -D -g \"\" hab")
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) startHabService(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	var command string
	if p.UseSudo {
		command = fmt.Sprintf("env HAB_NONINTERACTIVE=true sudo -E hab pkg install %s", service.Name)
	} else {
		command = fmt.Sprintf("env HAB_NONINTERACTIVE=true hab pkg install %s", service.Name)
	}
	err := p.runCommand(o, comm, command)
	if err != nil {
		return err
	}

	err = p.uploadUserTOML(o, comm, service)
	if err != nil {
		return err
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
	command = fmt.Sprintf("hab sup start %s %s", service.Name, options)
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	return p.runCommand(o, comm, command)
}

func (p *provisioner) uploadUserTOML(o terraform.UIOutput, comm communicator.Communicator, service Service) error {
	// Create the hab svc directory to lay down the user.toml before loading the service
	destDir := fmt.Sprintf("/hab/svc/%s", service.getPackageName(service.Name))
	command := fmt.Sprintf("mkdir -p %s", destDir)
	if p.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	err := p.runCommand(o, comm, command)
	if err != nil {
		return err
	}

	// Use tee to lay down user.toml instead of the communicator file uploader to get around permissions issues.
	command = fmt.Sprintf("sudo echo '%s' | sudo tee %s > /dev/null", service.UserTOML, path.Join(destDir, "user.toml"))
	fmt.Println("Command: " + command)
	o.Output("Command: " + command)
	if p.UseSudo {
		command = fmt.Sprintf("sudo echo '%s' | sudo tee %s > /dev/null", service.UserTOML, path.Join(destDir, "user.toml"))
	} else {
		command = fmt.Sprintf("echo '%s' | tee %s > /dev/null", service.UserTOML, path.Join(destDir, "user.toml"))
	}
	return p.runCommand(o, comm, command)
}

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
