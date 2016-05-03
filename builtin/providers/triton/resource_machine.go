package triton

import (
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/gosdc/cloudapi"
)

var (
	machineStateRunning = "running"
	machineStateStopped = "stopped"
	machineStateDeleted = "deleted"

	machineStateChangeTimeout       = 10 * time.Minute
	machineStateChangeCheckInterval = 10 * time.Second

	resourceMachineMetadataKeys = map[string]string{
		// semantics: "schema_name": "metadata_name"
		"root_authorized_keys": "root_authorized_keys",
		"user_script":          "user-script",
		"user_data":            "user-data",
		"administrator_pw":     "administrator-pw",
	}
)

func resourceMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceMachineCreate,
		Exists: resourceMachineExists,
		Read:   resourceMachineRead,
		Update: resourceMachineUpdate,
		Delete: resourceMachineDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description:  "friendly name",
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: resourceMachineValidateName,
			},
			"type": {
				Description: "machine type (smartmachine or virtualmachine)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"state": {
				Description: "current state of the machine",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dataset": {
				Description: "dataset URN the machine was provisioned with",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"memory": {
				Description: "amount of memory the machine has (in Mb)",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"disk": {
				Description: "amount of disk the machine has (in Gb)",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"ips": {
				Description: "IP addresses the machine has",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Description: "machine tags",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"created": {
				Description: "when the machine was created",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"updated": {
				Description: "when the machine was update",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"package": {
				Description: "name of the package to use on provisioning",
				Type:        schema.TypeString,
				Required:    true,
			},
			"image": {
				Description: "image UUID",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				// TODO: validate that the UUID is valid
			},
			"primaryip": {
				Description: "the primary (public) IP address for the machine",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"nic": {
				Description: "network interface",
				Type:        schema.TypeSet,
				Computed:    true,
				Optional:    true,
				Set: func(v interface{}) int {
					m := v.(map[string]interface{})
					return hashcode.String(m["network"].(string))
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Description: "NIC's IPv4 address",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"mac": {
							Description: "NIC's MAC address",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"primary": {
							Description: "Whether this is the machine's primary NIC",
							Computed:    true,
							Type:        schema.TypeBool,
						},
						"netmask": {
							Description: "IPv4 netmask",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"gateway": {
							Description: "IPv4 gateway",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"state": {
							Description: "describes the state of the NIC (e.g. provisioning, running, or stopped)",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"network": {
							Description: "Network ID this NIC is attached to",
							Required:    true,
							Type:        schema.TypeString,
						},
					},
				},
			},
			"firewall_enabled": {
				Description: "enable firewall for this machine",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},

			// computed resources from metadata
			"root_authorized_keys": {
				Description: "authorized keys for the root user on this machine",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"user_script": {
				Description: "user script to run on boot (every boot on SmartMachines)",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"user_data": {
				Description: "copied to machine on boot",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"administrator_pw": {
				Description: "administrator's initial password (Windows only)",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},

			// deprecated fields
			"networks": {
				Description: "desired network IDs",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Deprecated:  "Networks is deprecated, please use `nic`",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceMachineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	var networks []string
	for _, network := range d.Get("networks").([]interface{}) {
		networks = append(networks, network.(string))
	}
	nics := d.Get("nic").(*schema.Set)
	for _, nicI := range nics.List() {
		nic := nicI.(map[string]interface{})
		networks = append(networks, nic["network"].(string))
	}

	metadata := map[string]string{}
	for schemaName, metadataKey := range resourceMachineMetadataKeys {
		if v, ok := d.GetOk(schemaName); ok {
			metadata[metadataKey] = v.(string)
		}
	}

	tags := map[string]string{}
	for k, v := range d.Get("tags").(map[string]interface{}) {
		tags[k] = v.(string)
	}

	machine, err := client.CreateMachine(cloudapi.CreateMachineOpts{
		Name:            d.Get("name").(string),
		Package:         d.Get("package").(string),
		Image:           d.Get("image").(string),
		Networks:        networks,
		Metadata:        metadata,
		Tags:            tags,
		FirewallEnabled: d.Get("firewall_enabled").(bool),
	})
	if err != nil {
		return err
	}

	err = waitForMachineState(client, machine.Id, machineStateRunning, machineStateChangeTimeout)
	if err != nil {
		return err
	}

	// refresh state after it provisions
	d.SetId(machine.Id)
	err = resourceMachineRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceMachineExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*cloudapi.Client)

	machine, err := client.GetMachine(d.Id())

	return machine != nil && err == nil, err
}

func resourceMachineRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	machine, err := client.GetMachine(d.Id())
	if err != nil {
		return err
	}

	nics, err := client.ListNICs(d.Id())
	if err != nil {
		return err
	}

	d.SetId(machine.Id)
	d.Set("name", machine.Name)
	d.Set("type", machine.Type)
	d.Set("state", machine.State)
	d.Set("dataset", machine.Dataset)
	d.Set("memory", machine.Memory)
	d.Set("disk", machine.Disk)
	d.Set("ips", machine.IPs)
	d.Set("tags", machine.Tags)
	d.Set("created", machine.Created)
	d.Set("updated", machine.Updated)
	d.Set("package", machine.Package)
	d.Set("image", machine.Image)
	d.Set("primaryip", machine.PrimaryIP)
	d.Set("firewall_enabled", machine.FirewallEnabled)

	// create and update NICs
	var (
		machineNICs []map[string]interface{}
		networks    []string
	)
	for _, nic := range nics {
		machineNICs = append(
			machineNICs,
			map[string]interface{}{
				"ip":      nic.IP,
				"mac":     nic.MAC,
				"primary": nic.Primary,
				"netmask": nic.Netmask,
				"gateway": nic.Gateway,
				"state":   nic.State,
				"network": nic.Network,
			},
		)
		networks = append(networks, nic.Network)
	}
	d.Set("nic", machineNICs)
	d.Set("networks", networks)

	// computed attributes from metadata
	for schemaName, metadataKey := range resourceMachineMetadataKeys {
		d.Set(schemaName, machine.Metadata[metadataKey])
	}

	return nil
}

func resourceMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	d.Partial(true)

	if d.HasChange("name") {
		if err := client.RenameMachine(d.Id(), d.Get("name").(string)); err != nil {
			return err
		}

		err := waitFor(
			func() (bool, error) {
				machine, err := client.GetMachine(d.Id())
				return machine.Name == d.Get("name").(string), err
			},
			machineStateChangeCheckInterval,
			1*time.Minute,
		)
		if err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.HasChange("tags") {
		tags := map[string]string{}
		for k, v := range d.Get("tags").(map[string]interface{}) {
			tags[k] = v.(string)
		}

		var err error
		if len(tags) == 0 {
			err = client.DeleteMachineTags(d.Id())
		} else {
			_, err = client.ReplaceMachineTags(d.Id(), tags)
		}
		if err != nil {
			return err
		}

		err = waitFor(
			func() (bool, error) {
				machine, err := client.GetMachine(d.Id())
				return reflect.DeepEqual(machine.Tags, tags), err
			},
			machineStateChangeCheckInterval,
			1*time.Minute,
		)
		if err != nil {
			return err
		}

		d.SetPartial("tags")
	}

	if d.HasChange("package") {
		if err := client.ResizeMachine(d.Id(), d.Get("package").(string)); err != nil {
			return err
		}

		err := waitFor(
			func() (bool, error) {
				machine, err := client.GetMachine(d.Id())
				return machine.Package == d.Get("package").(string) && machine.State == machineStateRunning, err
			},
			machineStateChangeCheckInterval,
			machineStateChangeTimeout,
		)
		if err != nil {
			return err
		}

		d.SetPartial("package")
	}

	if d.HasChange("firewall_enabled") {
		var err error
		if d.Get("firewall_enabled").(bool) {
			err = client.EnableFirewallMachine(d.Id())
		} else {
			err = client.DisableFirewallMachine(d.Id())
		}
		if err != nil {
			return err
		}

		err = waitFor(
			func() (bool, error) {
				machine, err := client.GetMachine(d.Id())
				return machine.FirewallEnabled == d.Get("firewall_enabled").(bool), err
			},
			machineStateChangeCheckInterval,
			machineStateChangeTimeout,
		)

		if err != nil {
			return err
		}

		d.SetPartial("firewall_enabled")
	}

	if d.HasChange("nic") {
		o, n := d.GetChange("nic")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		oldNICs := o.(*schema.Set)
		newNICs := o.(*schema.Set)

		// add new NICs that are not in old NICs
		for _, nicI := range newNICs.Difference(oldNICs).List() {
			nic := nicI.(map[string]interface{})
			fmt.Printf("adding %+v\n", nic)
			_, err := client.AddNIC(d.Id(), nic["network"].(string))
			if err != nil {
				return err
			}
		}

		// remove old NICs that are not in new NICs
		for _, nicI := range oldNICs.Difference(newNICs).List() {
			nic := nicI.(map[string]interface{})
			fmt.Printf("removing %+v\n", nic)
			err := client.RemoveNIC(d.Id(), nic["mac"].(string))
			if err != nil {
				return err
			}
		}

		d.SetPartial("nic")
	}

	// metadata stuff
	metadata := map[string]string{}
	for schemaName, metadataKey := range resourceMachineMetadataKeys {
		if d.HasChange(schemaName) {
			metadata[metadataKey] = d.Get(schemaName).(string)
		}
	}
	if len(metadata) > 0 {
		_, err := client.UpdateMachineMetadata(d.Id(), metadata)
		if err != nil {
			return err
		}

		err = waitFor(
			func() (bool, error) {
				machine, err := client.GetMachine(d.Id())
				for k, v := range metadata {
					if provider_v, ok := machine.Metadata[k]; !ok || v != provider_v {
						return false, err
					}
				}
				return true, err
			},
			machineStateChangeCheckInterval,
			1*time.Minute,
		)
		if err != nil {
			return err
		}

		for schemaName := range resourceMachineMetadataKeys {
			if d.HasChange(schemaName) {
				d.SetPartial(schemaName)
			}
		}
	}

	d.Partial(false)

	err := resourceMachineRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceMachineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	state, err := readMachineState(client, d.Id())
	if state != machineStateStopped {
		err = client.StopMachine(d.Id())
		if err != nil {
			return err
		}

		waitForMachineState(client, d.Id(), machineStateStopped, machineStateChangeTimeout)
	}

	err = client.DeleteMachine(d.Id())
	if err != nil {
		return err
	}

	waitForMachineState(client, d.Id(), machineStateDeleted, machineStateChangeTimeout)
	return nil
}

func readMachineState(api *cloudapi.Client, id string) (string, error) {
	machine, err := api.GetMachine(id)
	if err != nil {
		return "", err
	}

	return machine.State, nil
}

// waitForMachineState waits for a machine to be in the desired state (waiting
// some seconds between each poll). If it doesn't reach the state within the
// duration specified in `timeout`, it returns ErrMachineStateTimeout.
func waitForMachineState(api *cloudapi.Client, id, state string, timeout time.Duration) error {
	return waitFor(
		func() (bool, error) {
			currentState, err := readMachineState(api, id)
			return currentState == state, err
		},
		machineStateChangeCheckInterval,
		machineStateChangeTimeout,
	)
}

func resourceMachineValidateName(value interface{}, name string) (warnings []string, errors []error) {
	warnings = []string{}
	errors = []error{}

	r := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\_\.\-]*$`)
	if !r.Match([]byte(value.(string))) {
		errors = append(errors, fmt.Errorf(`"%s" is not a valid %s`, value.(string), name))
	}

	return warnings, errors
}
