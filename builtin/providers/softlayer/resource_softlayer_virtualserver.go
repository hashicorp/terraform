package softlayer

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

func resourceSoftLayerVirtualserver() *schema.Resource {
	return &schema.Resource{
		Create: resourceSoftLayerVirtualserverCreate,
		Read: resourceSoftLayerVirtualserverRead,
		Update: resourceSoftLayerVirtualserverUpdate,
		Delete: resourceSoftLayerVirtualserverDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cpu": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"ram": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"public_network_speed": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default: 1000,
			},

			"ipv4_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv4_address_private": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
		},
	}
}

func resourceSoftLayerVirtualserverCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	dc := datatypes.Datacenter {
		Name: d.Get("region").(string),
	}

	networkComponent := datatypes.NetworkComponents {
		MaxSpeed: d.Get("public_network_speed").(int),
	}

	opts := datatypes.SoftLayer_Virtual_Guest_Template {
		Hostname: d.Get("name").(string),
		Domain: d.Get("domain").(string),
		OperatingSystemReferenceCode: d.Get("image").(string),
		HourlyBillingFlag: true,
		Datacenter: dc,
		StartCpus: d.Get("cpu").(int),
		MaxMemory: d.Get("ram").(int),
		NetworkComponents: []datatypes.NetworkComponents{networkComponent},
	}

	// Get configured ssh_keys
	ssh_keys := d.Get("ssh_keys.#").(int)
	if ssh_keys > 0 {
		opts.SshKeys = make([]datatypes.SshKey, 0, ssh_keys)
		for i := 0; i < ssh_keys; i++ {
			key := fmt.Sprintf("ssh_keys.%d", i)
			id := d.Get(key).(int)
			sshKey := datatypes.SshKey {
			  Id: id,
			}
			opts.SshKeys = append(opts.SshKeys, sshKey)
		}
	}

	log.Printf("[INFO] Creating virtual machine")

	guest, err := client.CreateObject(opts)

	if err != nil {
		return fmt.Errorf("Error creating virtual server: %s", err)
	}

	d.SetId(fmt.Sprintf("%d", guest.Id))

	log.Printf("[INFO] Virtual Machine ID: %s", d.Id())

	// wait for machine availability
	_, err = WaitForNoActiveTransactions(d, meta)

	if err != nil {
		return fmt.Errorf(
			"Error waiting for virtual machine (%s) to become ready: %s", d.Id(), err)
	}

	_, err = WaitForPublicIpAvailable(d, meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for virtual machine (%s) to become ready: %s", d.Id(), err)
	}

	return resourceSoftLayerVirtualserverRead(d, meta)
}

func resourceSoftLayerVirtualserverRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}
	result, err := client.GetObject(id)
	if err != nil {
		return fmt.Errorf("Error retrieving virtual server: %s", err)
	}

	d.Set("name", result.Hostname)
	d.Set("domain", result.Domain)
	if result.Datacenter != nil {
		d.Set("region", result.Datacenter.Name)
	}
	d.Set("public_network_speed", result.NetworkComponents[0].MaxSpeed)
	d.Set("cpu", result.StartCpus)
	d.Set("ram", result.MaxMemory)
	d.Set("has_public_ip", result.PrimaryIpAddress != "")
	d.Set("ipv4_address", result.PrimaryIpAddress)
	d.Set("ipv4_address_private", result.PrimaryBackendIpAddress)
	return nil
}

func resourceSoftLayerVirtualserverUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}
	result, err := client.GetObject(id)
	if err != nil {
		return fmt.Errorf("Error retrieving virtual server: %s", err)
	}

	result.Hostname = d.Get("name").(string)
	result.Domain = d.Get("domain").(string)
	result.StartCpus = d.Get("cpu").(int)
	result.MaxMemory = d.Get("ram").(int)
	result.NetworkComponents[0].MaxSpeed = d.Get("public_network_speed").(int)

	_, err = client.EditObject(id, result)

	if err != nil {
		return fmt.Errorf("Couldn't update virtual server: %s", err)
	}

	return nil
}

func resourceSoftLayerVirtualserverDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	_, err = WaitForNoActiveTransactions(d, meta)

	if err != nil {
		return fmt.Errorf("Error deleting virtual server, couldn't wait for zero active transactions: %s", err)
	}

	_, err = client.DeleteObject(id)

	if err != nil {
		return fmt.Errorf("Error deleting virtual server: %s", err)
	}

	return nil
}

func WaitForPublicIpAvailable(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to get a public IP", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"", "unavailable"},
		Target: "available",
		Refresh: func() (interface{}, string, error) {
			fmt.Println("Refreshing server state...")
			client := meta.(*Client).virtualGuestService
			id, err := strconv.Atoi(d.Id())
			if err != nil {
				return nil, "", fmt.Errorf("Not a valid ID, must be an integer: %s", err)
			}
			result, err := client.GetObject(id)
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving virtual server: %s", err)
			}
			if result.PrimaryIpAddress == "" {
				return result, "unavailable", nil
			} else {
				return result, "available", nil
			}
		},
		Timeout: 30 * time.Minute,
		Delay: 10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func WaitForNoActiveTransactions(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to have zero active transactions", d.Id())
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("The instance ID %s must be numeric", d.Id())
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"", "active"},
		Target: "idle",
		Refresh: func() (interface{}, string, error) {
			client := meta.(*Client).virtualGuestService
			transactions, err := client.GetActiveTransactions(id)
			if err != nil {
				return nil, "", fmt.Errorf("Couldn't get active transactions: %s", err)
			}
			if len(transactions) == 0 {
				return transactions, "idle", nil
			} else {
				return transactions, "active", nil
			}
		},
		Timeout: 10 * time.Minute,
		Delay: 10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}
