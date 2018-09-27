package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetworkingSecGroupV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingSecGroupV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"secgroup_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

func dataSourceNetworkingSecGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))

	listOpts := groups.ListOpts{
		ID:       d.Get("secgroup_id").(string),
		Name:     d.Get("name").(string),
		TenantID: d.Get("tenant_id").(string),
	}

	pages, err := groups.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return err
	}

	allSecGroups, err := groups.ExtractGroups(pages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve security groups: %s", err)
	}

	if len(allSecGroups) < 1 {
		return fmt.Errorf("No Security Group found with name: %s", d.Get("name"))
	}

	if len(allSecGroups) > 1 {
		return fmt.Errorf("More than one Security Group found with name: %s", d.Get("name"))
	}

	secGroup := allSecGroups[0]

	log.Printf("[DEBUG] Retrieved Security Group %s: %+v", secGroup.ID, secGroup)
	d.SetId(secGroup.ID)

	d.Set("name", secGroup.Name)
	d.Set("description", secGroup.Description)
	d.Set("tenant_id", secGroup.TenantID)
	d.Set("region", GetRegion(d, config))

	return nil
}
