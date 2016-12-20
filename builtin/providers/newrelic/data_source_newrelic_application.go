package newrelic

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	newrelic "github.com/paultyng/go-newrelic/api"
)

func dataSourceNewRelicApplication() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNewRelicApplicationRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"instance_ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeInt},
				Computed: true,
			},
			"host_ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeInt},
				Computed: true,
			},
		},
	}
}

func dataSourceNewRelicApplicationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	log.Printf("[INFO] Reading New Relic applications")

	applications, err := client.ListApplications()
	if err != nil {
		return err
	}

	var application *newrelic.Application
	name := d.Get("name").(string)

	for _, a := range applications {
		if a.Name == name {
			application = &a
			break
		}
	}

	if application == nil {
		return fmt.Errorf("The name '%s' does not match any New Relic applications.", name)
	}

	d.SetId(strconv.Itoa(application.ID))
	d.Set("name", application.Name)
	d.Set("instance_ids", application.Links.InstanceIDs)
	d.Set("host_ids", application.Links.HostIDs)

	return nil
}
