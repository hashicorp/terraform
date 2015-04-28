package sdc

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/kiasaki/go-sdc"
)

func resourceSDCMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceSDCMachineCreate,
		Read:   resourceSDCMachineRead,
		Update: resourceSDCMachineUpdate,
		Delete: resourceSDCMachineDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"package": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"networks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"default_networks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"firewall_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			// Computed properties
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ips": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"memory": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"disk": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSDCMachineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*sdc.Client)

	// Build up our creation options
	opts := &sdc.CreateMachineRequest{
		Name:    d.Get("name").(string),
		Image:   d.Get("image").(string),
		Package: d.Get("package").(string),
	}

	// Add optionnal parameters if present
	if items, ok := stringToStringMapFromResourceData(d, "metadata"); ok {
		opts.Metadata = items
	}
	if items, ok := stringToStringMapFromResourceData(d, "tags"); ok {
		opts.Tags = items
	}

	if attr, ok := d.GetOk("firewall_enabled"); ok {
		opts.FirewallEnabled = attr.(bool)
	}

	// Get configured networks
	if list, ok := stringListFromResourceData(d, "networks"); ok {
		opts.Networks = list
	}
	// Get configured default_networks
	if list, ok := stringListFromResourceData(d, "default_networks"); ok {
		opts.DefaultNetworks = list
	}

	log.Printf("[DEBUG] Machine create configuration: %#v", opts)

	machine, err := client.CreateMachine(opts)
	if err != nil {
		return fmt.Errorf("Error creating machine: %s", err)
	}

	// Assign the machine's id
	d.SetId(machine.Id)

	log.Printf("[INFO] Machine ID: %s", d.Id())

	_, err = WaitForMachineAttribute(d, "running", []string{"provisioning"}, "state", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for machine (%s) to become ready: %s", d.Id(), err)
	}

	return resourceSDCMachineRead(d, meta)
}

func resourceSDCMachineRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*sdc.Client)

	// Retrieve the machine properties for updating the state
	machine, err := client.GetMachine(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving machine: %s", err)
	}

	d.Set("name", machine.Name)
	d.Set("image", machine.Image)
	d.Set("package", machine.Package)
	d.Set("state", machine.State)
	d.Set("type", machine.Type)

	d.Set("memory", machine.Memory)
	d.Set("disk", machine.Disk)
	d.Set("ips", machine.Ips)

	// Initialize the connection info
	if len(machine.Ips) > 0 {
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": machine.Ips[0],
		})
	}

	return nil
}

func resourceSDCMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceSDCMachineRead(d, meta)
}

func resourceSDCMachineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*sdc.Client)

	// If machine is already deleted, we have nothing left to do
	if d.Get("state").(string) == "deleted" {
		return nil
	}

	// Start by stopping the machine as SDC doesn't allow deleting a running
	// machine
	err := client.StopMachine(d.Id())
	// Handle remotely destroyed machines
	if err != nil && strings.Contains(err.Error(), "ResourceNotFound") {
		return nil
	} else if err != nil {
		return fmt.Errorf(
			"Error stopping machine for destroy (%s): %s", d.Id(), err)
	}

	// Wait until stop operation has completed
	_, err = WaitForMachineAttribute(d, "stopped", []string{"stopping", "running"}, "state", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for machine to be stopped for destroy (%s): %s", d.Id(), err)
	}

	log.Printf("[INFO] Deleting machine: %s", d.Id())

	// Destroy the machine
	err = client.DeleteMachine(d.Id())
	// Handle remotely destroyed machines
	if err != nil && strings.Contains(err.Error(), "ResourceNotFound") {
		return nil
	} else if err != nil {
		return fmt.Errorf("Error deleting machine: %s", err)
	}

	return nil
}

func WaitForMachineAttribute(
	d *schema.ResourceData,
	target string,
	pending []string,
	attribute string,
	meta interface{},
) (interface{}, error) {
	log.Printf(
		"[INFO] Waiting for machine (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    machineStateRefreshFunc(d, attribute, meta),
		Timeout:    60 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func machineStateRefreshFunc(
	d *schema.ResourceData, attribute string, meta interface{},
) resource.StateRefreshFunc {
	client := meta.(*sdc.Client)

	return func() (interface{}, string, error) {
		err := resourceSDCMachineRead(d, meta)
		if err != nil {
			return nil, "", err
		}

		// See if we can access our attribute
		if attr, ok := d.GetOk(attribute); ok {
			// Retrieve the machine properties
			machine, err := client.GetMachine(d.Id())
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving machine: %s", err)
			}

			return &machine, attr.(string), nil
		}

		return nil, "", fmt.Errorf("Can't wait for change on unknown attribute '%s'", attribute)
	}
}

// Extracts a string array from a ResourceData instance at a certain key, if present.
func stringListFromResourceData(d *schema.ResourceData, key string) ([]string, bool) {
	list := []string{}

	length := d.Get(key + ".#").(int)
	if length > 0 {
		for i := 0; i < length; i++ {
			itemKey := fmt.Sprintf(key+".%d", i)
			list = append(list, d.Get(itemKey).(string))
		}
		return list, true
	}

	return nil, false
}

func stringToStringMapFromResourceData(d *schema.ResourceData, key string) (map[string]string, bool) {
	if attr, ok := d.GetOk(key); ok {
		items := attr.(map[string]interface{})
		extractedItems := map[string]string{}

		for key, value := range items {
			extractedItems[key] = value.(string)
		}

		return extractedItems, true
	}

	return nil, false
}
