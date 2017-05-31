package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRegions() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRegionsRead,

		Schema: map[string]*schema.Schema{
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsRegionsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Fetching regions.")
	d.SetId(time.Now().UTC().String())

	request := &ec2.DescribeRegionsInput{}

	resp, err := conn.DescribeRegions(request)
	if err != nil {
		return fmt.Errorf("Error fetching regions: %s", err)
	}

	raw := make([]string, len(resp.Regions))
	for i, v := range resp.Regions {
		raw[i] = *v.RegionName
	}

	sort.Strings(raw)

	if err := d.Set("names", raw); err != nil {
		return fmt.Errorf("[WARN] Error setting regions: %s", err)
	}

	return nil
}
