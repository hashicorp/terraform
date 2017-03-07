package google

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	compute "google.golang.org/api/compute/v1"
)

func dataSourceGoogleComputeZones() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleComputeZonesRead,
		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if value != "UP" && value != "DOWN" {
						es = append(es, fmt.Errorf("%q can only be 'UP' or 'DOWN' (%q given)", k, value))
					}
					return
				},
			},
		},
	}
}

func dataSourceGoogleComputeZonesRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region := config.Region
	if r, ok := d.GetOk("region"); ok {
		region = r.(string)
	}

	regionUrl := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s",
		config.Project, region)
	filter := fmt.Sprintf("(region eq %s)", regionUrl)

	if s, ok := d.GetOk("status"); ok {
		filter += fmt.Sprintf(" (status eq %s)", s)
	}

	call := config.clientCompute.Zones.List(config.Project).Filter(filter)

	resp, err := call.Do()
	if err != nil {
		return err
	}

	zones := flattenZones(resp.Items)
	log.Printf("[DEBUG] Received Google Compute Zones: %q", zones)

	d.Set("names", zones)
	d.SetId(time.Now().UTC().String())

	return nil
}

func flattenZones(zones []*compute.Zone) []string {
	result := make([]string, len(zones), len(zones))
	for i, zone := range zones {
		result[i] = zone.Name
	}
	sort.Strings(result)
	return result
}
