package ovh

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePublicCloudRegions() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePublicCloudRegionsRead,
		Schema: map[string]*schema.Schema{
			"project_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_PROJECT_ID", nil),
			},
			"names": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourcePublicCloudRegionsRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectId := d.Get("project_id").(string)

	log.Printf("[DEBUG] Will read public cloud regions for project: %s", projectId)
	d.Partial(true)

	endpoint := fmt.Sprintf("/cloud/project/%s/region", projectId)
	names := make([]string, 0)
	err := config.OVHClient.Get(endpoint, &names)

	if err != nil {
		return fmt.Errorf("Error calling %s:\n\t %q", endpoint, err)
	}

	d.Set("names", names)
	d.Partial(false)
	d.SetId(projectId)

	log.Printf("[DEBUG] Read Public Cloud Regions %s", names)
	return nil
}
