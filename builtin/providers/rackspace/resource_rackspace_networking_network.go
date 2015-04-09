package rackspace

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	osNetworks "github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	rsNetworks "github.com/rackspace/gophercloud/rackspace/networking/v2/networks"
)

func resourceNetworkingNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingNetworkCreate,
		Read:   resourceNetworkingNetworkRead,
		Update: resourceNetworkingNetworkUpdate,
		Delete: resourceNetworkingNetworkDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("RS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"shared": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNetworkingNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	createOpts := osNetworks.CreateOpts{
		Name:     d.Get("name").(string),
		TenantID: d.Get("tenant_id").(string),
	}

	asuRaw := d.Get("admin_state_up").(string)
	if asuRaw != "" {
		asu, err := strconv.ParseBool(asuRaw)
		if err != nil {
			return fmt.Errorf("admin_state_up, if provided, must be either 'true' or 'false'")
		}
		createOpts.AdminStateUp = &asu
	}

	sharedRaw := d.Get("shared").(string)
	if sharedRaw != "" {
		shared, err := strconv.ParseBool(sharedRaw)
		if err != nil {
			return fmt.Errorf("shared, if provided, must be either 'true' or 'false': %v", err)
		}
		createOpts.Shared = &shared
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := rsNetworks.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace Neutron network: %s", err)
	}
	log.Printf("[INFO] Network ID: %s", n.ID)

	d.SetId(n.ID)

	return resourceNetworkingNetworkRead(d, meta)
}

func resourceNetworkingNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	n, err := rsNetworks.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "network")
	}

	log.Printf("[DEBUG] Retreived Network %s: %+v", d.Id(), n)

	d.Set("name", n.Name)
	d.Set("admin_state_up", strconv.FormatBool(n.AdminStateUp))
	d.Set("shared", strconv.FormatBool(n.Shared))
	d.Set("tenant_id", n.TenantID)

	return nil
}

func resourceNetworkingNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	var updateOpts osNetworks.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("admin_state_up") {
		asuRaw := d.Get("admin_state_up").(string)
		if asuRaw != "" {
			asu, err := strconv.ParseBool(asuRaw)
			if err != nil {
				return fmt.Errorf("admin_state_up, if provided, must be either 'true' or 'false'")
			}
			updateOpts.AdminStateUp = &asu
		}
	}
	if d.HasChange("shared") {
		sharedRaw := d.Get("shared").(string)
		if sharedRaw != "" {
			shared, err := strconv.ParseBool(sharedRaw)
			if err != nil {
				return fmt.Errorf("shared, if provided, must be either 'true' or 'false': %v", err)
			}
			updateOpts.Shared = &shared
		}
	}

	log.Printf("[DEBUG] Updating Network %s with options: %+v", d.Id(), updateOpts)

	_, err = rsNetworks.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Rackspace Neutron Network: %s", err)
	}

	return resourceNetworkingNetworkRead(d, meta)
}

func resourceNetworkingNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	err = rsNetworks.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace Neutron Network: %s", err)
	}

	d.SetId("")
	return nil
}
