package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
)

func resourceNetworkingNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingNetworkCreate,
		Read:   resourceNetworkingNetworkRead,
		Update: resourceNetworkingNetworkUpdate,
		Delete: resourceNetworkingNetworkDelete,

		Schema: map[string]*schema.Schema{
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
	osClient := config.networkingV2Client

	createOpts := networks.CreateOpts{
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

	log.Printf("[INFO] Requesting network creation")
	n, err := networks.Create(osClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron network: %s", err)
	}
	log.Printf("[INFO] Network ID: %s", n.ID)

	d.SetId(n.ID)

	return resourceNetworkingNetworkRead(d, meta)
}

func resourceNetworkingNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	n, err := networks.Get(osClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack Neutron Network: %s", err)
	}

	log.Printf("[DEBUG] Retreived Network %s: %+v", d.Id(), n)

	if _, exists := d.GetOk("name"); exists {
		if d.HasChange("name") {
			d.Set("name", n.Name)
		}
	} else {
		d.Set("name", "")
	}

	if _, exists := d.GetOk("admin_state_up"); exists {
		if d.HasChange("admin_state_up") {
			d.Set("admin_state_up", strconv.FormatBool(n.AdminStateUp))
		}
	} else {
		d.Set("admin_state_up", "")
	}

	if _, exists := d.GetOk("shared"); exists {
		if d.HasChange("shared") {
			d.Set("shared", strconv.FormatBool(n.Shared))
		}
	} else {
		d.Set("shared", "")
	}

	if _, exists := d.GetOk("tenant_id"); exists {
		if d.HasChange("tenant_id") {
			d.Set("tenant_id", n.TenantID)
		}
	} else {
		d.Set("tenant_id", "")
	}

	return nil
}

func resourceNetworkingNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	var updateOpts networks.UpdateOpts
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

	_, err := networks.Update(osClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Network: %s", err)
	}

	return resourceNetworkingNetworkRead(d, meta)
}

func resourceNetworkingNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	err := networks.Delete(osClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Network: %s", err)
	}

	d.SetId("")
	return nil
}
