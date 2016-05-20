package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/groups"
)

func resourceNetworkingSecGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSecGroupV2Create,
		Read:   resourceNetworkingSecGroupV2Read,
		Delete: resourceNetworkingSecGroupV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceNetworkingSecGroupV2Create(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := groups.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		TenantID:    d.Get("tenant_id").(string),
	}

	log.Printf("[DEBUG] Create OpenStack Neutron Security Group: %#v", opts)

	security_group, err := groups.Create(networkingClient, opts).Extract()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] OpenStack Neutron Security Group created: %#v", security_group)

	d.SetId(security_group.ID)

	return resourceNetworkingSecGroupV2Read(d, meta)
}

func resourceNetworkingSecGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about security group: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	security_group, err := groups.Get(networkingClient, d.Id()).Extract()

	if err != nil {
		return CheckDeleted(d, err, "OpenStack Neutron Security group")
	}

	d.Set("description", security_group.Description)
	d.Set("tenant_id", security_group.TenantID)
	return nil
}

func resourceNetworkingSecGroupV2Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy security group: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSecGroupDelete(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Security Group: %s", err)
	}

	d.SetId("")
	return err
}

func waitForSecGroupDelete(networkingClient *gophercloud.ServiceClient, secGroupId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack Security Group %s.\n", secGroupId)

		r, err := groups.Get(networkingClient, secGroupId).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Neutron Security Group %s", secGroupId)
				return r, "DELETED", nil
			}
		}

		err = groups.Delete(networkingClient, secGroupId).ExtractErr()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Neutron Security Group %s", secGroupId)
				return r, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Neutron Security Group %s still active.\n", secGroupId)
		return r, "ACTIVE", nil
	}
}
