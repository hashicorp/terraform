package google

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceGoogleContainerEngineVersions() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleContainerEngineVersionsRead,
		Schema: map[string]*schema.Schema{
			"project": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"zone": {
				Type:     schema.TypeString,
				Required: true,
			},
			"latest_master_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_node_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"valid_master_versions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"valid_node_versions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGoogleContainerEngineVersionsRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone := d.Get("zone").(string)

	resp, err := config.clientContainer.Projects.Zones.GetServerconfig(project, zone).Do()
	if err != nil {
		return fmt.Errorf("Error retrieving available container cluster versions: %s", err.Error())
	}

	d.Set("valid_master_versions", resp.ValidMasterVersions)
	d.Set("valid_node_versions", resp.ValidNodeVersions)
	d.Set("latest_master_version", resp.ValidMasterVersions[0])
	d.Set("latest_node_version", resp.ValidNodeVersions[0])

	d.SetId(time.Now().UTC().String())

	return nil
}
