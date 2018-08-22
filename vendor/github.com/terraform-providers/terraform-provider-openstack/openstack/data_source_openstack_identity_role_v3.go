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
			"domain_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
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

	log.Printf("[DEBUG] List Options: %#v", listOpts)

	var role roles.Role
	allPages, err := roles.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query roles: %s", err)
	}

	allRoles, err := roles.ExtractRoles(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve roles: %s", err)
	}

	if len(allRoles) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allRoles) > 1 {
		log.Printf("[DEBUG] Multiple results found: %#v", allRoles)
		return fmt.Errorf("Your query returned more than one result. Please try a more " +
			"specific search criteria, or set `most_recent` attribute to true.")
	}
	role = allRoles[0]

	log.Printf("[DEBUG] Single Role found: %s", role.ID)
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
