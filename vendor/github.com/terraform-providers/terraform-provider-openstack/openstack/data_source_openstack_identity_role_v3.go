package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceIdentityRoleV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityRoleV3Read,

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

// dataSourceIdentityRoleV3Read performs the role lookup.
func dataSourceIdentityRoleV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	listOpts := roles.ListOpts{
		DomainID: d.Get("domain_id").(string),
		Name:     d.Get("name").(string),
	}

	log.Printf("[DEBUG] openstack_identity_role_v3 list options: %#v", listOpts)

	var role roles.Role
	allPages, err := roles.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query openstack_identity_role_v3: %s", err)
	}

	allRoles, err := roles.ExtractRoles(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_identity_role_v3: %s", err)
	}

	if len(allRoles) < 1 {
		return fmt.Errorf("Your openstack_identity_role_v3 query returned no results.")
	}

	if len(allRoles) > 1 {
		return fmt.Errorf("Your openstack_identity_role_v3 query returned more than one result.")
	}

	role = allRoles[0]

	return dataSourceIdentityRoleV3Attributes(d, config, &role)
}

// dataSourceIdentityRoleV3Attributes populates the fields of an Role resource.
func dataSourceIdentityRoleV3Attributes(d *schema.ResourceData, config *Config, role *roles.Role) error {
	log.Printf("[DEBUG] openstack_identity_role_v3 details: %#v", role)

	d.SetId(role.ID)
	d.Set("name", role.Name)
	d.Set("domain_id", role.DomainID)
	d.Set("region", GetRegion(d, config))

	return nil
}
