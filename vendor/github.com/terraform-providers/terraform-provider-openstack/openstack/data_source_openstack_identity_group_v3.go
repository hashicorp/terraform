package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/groups"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceIdentityGroupV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityGroupV3Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"domain_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

// dataSourceIdentityGroupV3Read performs the group lookup.
func dataSourceIdentityGroupV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	listOpts := groups.ListOpts{
		DomainID: d.Get("domain_id").(string),
		Name:     d.Get("name").(string),
	}

	log.Printf("[DEBUG] openstack_identity_group_v3 list options: %#v", listOpts)

	var group groups.Group
	allPages, err := groups.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query openstack_identity_group_v3: %s", err)
	}

	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_identity_group_v3: %s", err)
	}

	if len(allGroups) < 1 {
		return fmt.Errorf("Your openstack_identity_group_v3 query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allGroups) > 1 {
		return fmt.Errorf("Your openstack_identity_group_v3 query returned more than one result.")
	}

	group = allGroups[0]

	return dataSourceIdentityGroupV3Attributes(d, config, &group)
}

// dataSourceIdentityRoleV3Attributes populates the fields of an Role resource.
func dataSourceIdentityGroupV3Attributes(d *schema.ResourceData, config *Config, group *groups.Group) error {
	log.Printf("[DEBUG] openstack_identity_group_v3 details: %#v", group)

	d.SetId(group.ID)
	d.Set("name", group.Name)
	d.Set("domain_id", group.DomainID)
	d.Set("region", GetRegion(d, config))

	return nil
}
