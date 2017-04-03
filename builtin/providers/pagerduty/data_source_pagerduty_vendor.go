package pagerduty

import (
	"fmt"
	"log"
	"regexp"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePagerDutyVendor() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyVendorRead,

		Schema: map[string]*schema.Schema{
			"name_regex": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `name` instead. This attribute will be removed in a future version",
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourcePagerDutyVendorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty vendor")

	searchName := d.Get("name").(string)

	o := &pagerduty.ListVendorOptions{
		Query: searchName,
	}

	resp, err := client.ListVendors(*o)
	if err != nil {
		return err
	}

	var found *pagerduty.Vendor

	r := regexp.MustCompile("(?i)" + searchName)

	for _, vendor := range resp.Vendors {
		if r.MatchString(vendor.Name) {
			found = &vendor
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any vendor with the name: %s", searchName)
	}

	d.SetId(found.ID)
	d.Set("name", found.Name)
	d.Set("type", found.GenericServiceType)

	return nil
}
