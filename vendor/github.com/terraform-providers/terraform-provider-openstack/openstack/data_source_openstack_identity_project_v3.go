package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceIdentityProjectV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityProjectV3Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"is_domain": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"parent_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

// dataSourceIdentityProjectV3Read performs the project lookup.
func dataSourceIdentityProjectV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	enabled := d.Get("enabled").(bool)
	isDomain := d.Get("is_domain").(bool)
	listOpts := projects.ListOpts{
		DomainID: d.Get("domain_id").(string),
		Enabled:  &enabled,
		IsDomain: &isDomain,
		Name:     d.Get("name").(string),
		ParentID: d.Get("parent_id").(string),
	}

	log.Printf("[DEBUG] openstack_identity_project_v3 list options: %#v", listOpts)

	var project projects.Project
	allPages, err := projects.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query openstack_identity_project_v3: %s", err)
	}

	allProjects, err := projects.ExtractProjects(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_identity_project_v3: %s", err)
	}

	if len(allProjects) < 1 {
		return fmt.Errorf("Your openstack_identity_project_v3 query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allProjects) > 1 {
		return fmt.Errorf("Your openstack_identity_project_v3 query returned more than one result.")
	}

	project = allProjects[0]

	return dataSourceIdentityProjectV3Attributes(d, &project)
}

// dataSourceIdentityProjectV3Attributes populates the fields of an Project resource.
func dataSourceIdentityProjectV3Attributes(d *schema.ResourceData, project *projects.Project) error {
	log.Printf("[DEBUG] openstack_identity_project_v3 details: %#v", project)

	d.SetId(project.ID)
	d.Set("is_domain", project.IsDomain)
	d.Set("description", project.Description)
	d.Set("domain_id", project.DomainID)
	d.Set("enabled", project.Enabled)
	d.Set("name", project.Name)
	d.Set("parent_id", project.ParentID)

	return nil
}
