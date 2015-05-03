package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
)

func resourceNetworkingPortV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingPortV2Create,
		Read:   resourceNetworkingPortV2Read,
		Update: resourceNetworkingPortV2Update,
		Delete: resourceNetworkingPortV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"mac_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"device_owner": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			"device_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNetworkingPortV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := ports.CreateOpts{
		Name:           d.Get("name").(string),
		AdminStateUp:   resourcePortAdminStateUpV2(d),
		NetworkID:      d.Get("network_id").(string),
		MACAddress:     d.Get("mac_address").(string),
		TenantID:       d.Get("tenant_id").(string),
		DeviceOwner:    d.Get("device_owner").(string),
		SecurityGroups: resourcePortSecurityGroupsV2(d),
		DeviceID:       d.Get("device_id").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	p, err := ports.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron network: %s", err)
	}
	log.Printf("[INFO] Network ID: %s", p.ID)

	d.SetId(p.ID)

	return resourceNetworkingPortV2Read(d, meta)
}

func resourceNetworkingPortV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	p, err := ports.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "port")
	}

	log.Printf("[DEBUG] Retreived Port %s: %+v", d.Id(), p)

	d.Set("name", p.Name)
	d.Set("admin_state_up", strconv.FormatBool(p.AdminStateUp))
	d.Set("network_id", p.NetworkID)
	d.Set("mac_address", p.MACAddress)
	d.Set("tenant_id", p.TenantID)
	d.Set("device_owner", p.DeviceOwner)
	d.Set("security_groups", p.SecurityGroups)
	d.Set("device_id", p.DeviceID)

	return nil
}

func resourceNetworkingPortV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts ports.UpdateOpts

	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}

	if d.HasChange("admin_state_up") {
		updateOpts.AdminStateUp = resourcePortAdminStateUpV2(d)
	}

	if d.HasChange("device_owner") {
		updateOpts.DeviceOwner = d.Get("device_owner").(string)
	}

	if d.HasChange("security_groups") {
		updateOpts.SecurityGroups = resourcePortSecurityGroupsV2(d)
	}

	if d.HasChange("device_id") {
		updateOpts.DeviceID = d.Get("device_id").(string)
	}

	log.Printf("[DEBUG] Updating Port %s with options: %+v", d.Id(), updateOpts)

	_, err = ports.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Network: %s", err)
	}

	return resourceNetworkingPortV2Read(d, meta)
}

func resourceNetworkingPortV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	err = ports.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Network: %s", err)
	}

	d.SetId("")
	return nil
}

func resourcePortSecurityGroupsV2(d *schema.ResourceData) []string {
	rawSecurityGroups := d.Get("security_groups").(*schema.Set)
	groups := make([]string, rawSecurityGroups.Len())
	for i, raw := range rawSecurityGroups.List() {
		groups[i] = raw.(string)
	}
	return groups
}

func resourcePortAdminStateUpV2(d *schema.ResourceData) *bool {
	value := false

	if raw, ok := d.GetOk("admin_state_up"); ok && raw == "true" {
		value = true
	}

	return &value
}
