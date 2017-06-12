package runscope

import (
	"fmt"
	"github.com/ewilde/go-runscope"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func dataSourceRunscopeIntegration() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRunscopeIntegrationRead,

		Schema: map[string]*schema.Schema{
			"team_uuid": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceRunscopeIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*runscope.Client)

	log.Printf("[INFO] Reading Runscope integration")

	searchType := d.Get("type").(string)

	resp, err := client.ListIntegrations(d.Get("team_uuid").(string))
	if err != nil {
		return err
	}

	found := &runscope.Integration{}
	for _, integration := range resp {
		if integration.IntegrationType == searchType {
			found = integration
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any user with the email: %s", searchType)
	}

	d.SetId(found.ID)
	d.Set("id", found.ID)
	d.Set("type", found.IntegrationType)

	return nil
}
