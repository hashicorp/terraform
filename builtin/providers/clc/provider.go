package clc

import (
	"fmt"
	"log"
	"strconv"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/api"
	"github.com/CenturyLinkCloud/clc-sdk/group"
	"github.com/CenturyLinkCloud/clc-sdk/server"
	"github.com/CenturyLinkCloud/clc-sdk/status"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider implements ResourceProvider for CLC
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLC_USERNAME", nil),
				Description: "Your CLC username",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLC_PASSWORD", nil),
				Description: "Your CLC password",
			},
			"account": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLC_ACCOUNT", ""),
				Description: "Account alias override",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"clc_server":             resourceCLCServer(),
			"clc_group":              resourceCLCGroup(),
			"clc_public_ip":          resourceCLCPublicIP(),
			"clc_load_balancer":      resourceCLCLoadBalancer(),
			"clc_load_balancer_pool": resourceCLCLoadBalancerPool(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	un := d.Get("username").(string)
	pw := d.Get("password").(string)

	config, err := api.NewConfig(un, pw)
	if err != nil {
		return nil, fmt.Errorf("Failed to create CLC config with provided details: %v", err)
	}
	config.UserAgent = fmt.Sprintf("terraform-clc terraform/%s", terraform.Version)
	// user requested alias override or sub-account
	if al := d.Get("account").(string); al != "" {
		config.Alias = al
	}

	client := clc.New(config)
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("Failed authenticated with provided credentials: %v", err)
	}

	alerts, err := client.Alert.GetAll()
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to the CLC api because %s", err)
	}
	for _, a := range alerts.Items {
		log.Printf("[WARN] Received alert: %v", a)
	}
	return client, nil
}

// package utility functions

func waitStatus(client *clc.Client, id string) error {
	// block until queue is processed and server is up
	poll := make(chan *status.Response, 1)
	err := client.Status.Poll(id, poll)
	if err != nil {
		return nil
	}
	status := <-poll
	log.Printf("[DEBUG] status %v", status)
	if status.Failed() {
		return fmt.Errorf("unsuccessful job %v failed with status: %v", id, status.Status)
	}
	return nil
}

func dcGroups(dcname string, client *clc.Client) (map[string]string, error) {
	dc, _ := client.DC.Get(dcname)
	_, id := dc.Links.GetID("group")
	m := map[string]string{}
	resp, _ := client.Group.Get(id)
	m[resp.Name] = resp.ID // top
	m[resp.ID] = resp.ID
	for _, x := range resp.Groups {
		deepGroups(x, &m)
	}
	return m, nil
}

func deepGroups(g group.Groups, m *map[string]string) {
	(*m)[g.Name] = g.ID
	(*m)[g.ID] = g.ID
	for _, sg := range g.Groups {
		deepGroups(sg, m)
	}
}

// resolveGroupByNameOrId takes a reference to a group (either name or guid)
// and returns the guid of the group
func resolveGroupByNameOrId(ref, dc string, client *clc.Client) (string, error) {
	m, err := dcGroups(dc, client)
	if err != nil {
		return "", fmt.Errorf("Failed pulling groups in location %v - %v", dc, err)
	}
	if id, ok := m[ref]; ok {
		return id, nil
	}
	return "", fmt.Errorf("Failed resolving group '%v' in location %v", ref, dc)
}

func stateFromString(st string) server.PowerState {
	switch st {
	case "on", "started":
		return server.On
	case "off", "stopped":
		return server.Off
	case "pause", "paused":
		return server.Pause
	case "reboot":
		return server.Reboot
	case "reset":
		return server.Reset
	case "shutdown":
		return server.ShutDown
	case "start_maintenance":
		return server.StartMaintenance
	case "stop_maintenance":
		return server.StopMaintenance
	}
	return -1
}

func parseCustomFields(d *schema.ResourceData) ([]api.Customfields, error) {
	var fields []api.Customfields
	if v := d.Get("custom_fields"); v != nil {
		for _, v := range v.([]interface{}) {
			m := v.(map[string]interface{})
			f := api.Customfields{
				ID:    m["id"].(string),
				Value: m["value"].(string),
			}
			fields = append(fields, f)
		}
	}
	return fields, nil
}

func parseAdditionalDisks(d *schema.ResourceData) ([]server.Disk, error) {
	// some complexity here: create has a different format than update
	// on-create: { path, sizeGB, type }
	// on-update: { diskId, sizeGB, (path), (type=partitioned) }
	var disks []server.Disk
	if v := d.Get("additional_disks"); v != nil {
		for _, v := range v.([]interface{}) {
			m := v.(map[string]interface{})
			ty := m["type"].(string)
			var pa string
			if nil != m["path"] {
				pa = m["path"].(string)
			}
			sz, err := strconv.Atoi(m["size_gb"].(string))
			if err != nil {
				log.Printf("[WARN] Failed parsing size '%v'. skipping", m["size_gb"])
				return nil, fmt.Errorf("Unable to parse %v as int", m["size_gb"])
			}
			if ty != "raw" && ty != "partitioned" {
				return nil, fmt.Errorf("Expected type of { raw | partitioned }. received %v", ty)
			}
			if ty == "raw" && pa != "" {
				return nil, fmt.Errorf("Path can not be specified for raw disks")
			}
			disk := server.Disk{
				SizeGB: sz,
				Type:   ty,
			}
			if pa != "" {
				disk.Path = pa
			}
			disks = append(disks, disk)
		}
	}
	return disks, nil
}
