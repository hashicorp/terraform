package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRegion() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRegionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"current": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsRegionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	currentRegion := meta.(*AWSClient).region

	req := &ec2.DescribeRegionsInput{}

	req.RegionNames = make([]*string, 0, 2)
	if name := d.Get("name").(string); name != "" {
		req.RegionNames = append(req.RegionNames, aws.String(name))
	}

	if d.Get("current").(bool) {
		req.RegionNames = append(req.RegionNames, aws.String(currentRegion))
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"endpoint": d.Get("endpoint").(string),
		},
	)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading Region: %s", req)
	resp, err := conn.DescribeRegions(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.Regions) == 0 {
		return fmt.Errorf("no matching regions found")
	}
	if len(resp.Regions) > 1 {
		return fmt.Errorf("multiple regions matched; use additional constraints to reduce matches to a single region")
	}

	region := resp.Regions[0]

	d.SetId(*region.RegionName)
	d.Set("name", region.RegionName)
	d.Set("endpoint", region.Endpoint)
	d.Set("current", *region.RegionName == currentRegion)

	return nil
}
