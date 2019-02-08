package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/addressscopes"
)

func resourceNetworkingAddressScopeV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingAddressScopeV2Create,
		Read:   resourceNetworkingAddressScopeV2Read,
		Update: resourceNetworkingAddressScopeV2Update,
		Delete: resourceNetworkingAddressScopeV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"ip_version": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  4,
				ForceNew: true,
			},

			"shared": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceNetworkingAddressScopeV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := addressscopes.CreateOpts{
		Name:      d.Get("name").(string),
		ProjectID: d.Get("project_id").(string),
		IPVersion: d.Get("ip_version").(int),
		Shared:    d.Get("shared").(bool),
	}

	log.Printf("[DEBUG] openstack_networking_addressscope_v2 create options: %#v", createOpts)
	a, err := addressscopes.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_networking_addressscope_v2: %s", err)
	}

	log.Printf("[DEBUG] Waiting for openstack_networking_addressscope_v2 %s to become available", a.ID)

	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Refresh:    resourceNetworkingAddressScopeV2StateRefreshFunc(networkingClient, a.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_networking_addressscope_v2 %s to become available: %s", a.ID, err)
	}

	d.SetId(a.ID)

	log.Printf("[DEBUG] Created openstack_networking_addressscope_v2 %s: %#v", a.ID, a)
	return resourceNetworkingAddressScopeV2Read(d, meta)
}

func resourceNetworkingAddressScopeV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	a, err := addressscopes.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error getting openstack_networking_addressscope_v2")
	}

	log.Printf("[DEBUG] Retrieved openstack_networking_addressscope_v2 %s: %#v", d.Id(), a)

	d.Set("region", GetRegion(d, config))
	d.Set("name", a.Name)
	d.Set("project_id", a.ProjectID)
	d.Set("ip_version", a.IPVersion)
	d.Set("shared", a.Shared)

	return nil
}

func resourceNetworkingAddressScopeV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var (
		hasChange  bool
		updateOpts addressscopes.UpdateOpts
	)

	if d.HasChange("name") {
		hasChange = true
		v := d.Get("name").(string)
		updateOpts.Name = &v
	}

	if d.HasChange("shared") {
		hasChange = true
		v := d.Get("shared").(bool)
		updateOpts.Shared = &v
	}

	if hasChange {
		log.Printf("[DEBUG] openstack_networking_addressscope_v2 %s update options: %#v", d.Id(), updateOpts)
		_, err = addressscopes.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating openstack_networking_addressscope_v2 %s: %s", d.Id(), err)
		}
	}

	return resourceNetworkingAddressScopeV2Read(d, meta)
}

func resourceNetworkingAddressScopeV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	if err := addressscopes.Delete(networkingClient, d.Id()).ExtractErr(); err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_networking_addressscope_v2")
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    resourceNetworkingAddressScopeV2StateRefreshFunc(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_networking_addressscope_v2 %s to delete: %s", d.Id(), err)
	}

	return nil
}
