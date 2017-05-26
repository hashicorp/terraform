package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"strings"
)

func dataSourceDataCenter() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDataCenterRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"location": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceDataCenterRead(d *schema.ResourceData, meta interface{}) error {
	datacenters := profitbricks.ListDatacenters()

	if datacenters.StatusCode > 299 {
		return fmt.Errorf("An error occured while fetching datacenters %s", datacenters.Response)
	}

	name := d.Get("name").(string)
	location, locationOk := d.GetOk("location")

	results := []profitbricks.Datacenter{}

	for _, dc := range datacenters.Items {
		if dc.Properties.Name == name || strings.Contains(dc.Properties.Name, name) {
			results = append(results, dc)
		}
	}

	if locationOk {
		log.Printf("[INFO] searching dcs by location***********")
		locationResults := []profitbricks.Datacenter{}
		for _, dc := range results {
			if dc.Properties.Location == location.(string) {
				locationResults = append(locationResults, dc)
			}
		}
		results = locationResults
	}
	log.Printf("[INFO] Results length %d *************", len(results))

	if len(results) > 1 {
		log.Printf("[INFO] Results length greater than 1")
		return fmt.Errorf("There is more than one datacenters that match the search criteria")
	}

	if len(results) == 0 {
		return fmt.Errorf("There are no datacenters that match the search criteria")
	}

	d.SetId(results[0].Id)

	return nil
}
