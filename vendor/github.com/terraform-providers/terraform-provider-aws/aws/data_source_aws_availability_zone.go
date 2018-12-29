package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAvailabilityZone() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAvailabilityZoneRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name_suffix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"zone_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsAvailabilityZoneRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeAvailabilityZonesInput{}

	if v := d.Get("name").(string); v != "" {
		req.ZoneNames = []*string{aws.String(v)}
	}
	if v := d.Get("zone_id").(string); v != "" {
		req.ZoneIds = []*string{aws.String(v)}
	}
	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"state": d.Get("state").(string),
		},
	)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading Availability Zone: %s", req)
	resp, err := conn.DescribeAvailabilityZones(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.AvailabilityZones) == 0 {
		return fmt.Errorf("no matching AZ found")
	}
	if len(resp.AvailabilityZones) > 1 {
		return fmt.Errorf("multiple AZs matched; use additional constraints to reduce matches to a single AZ")
	}

	az := resp.AvailabilityZones[0]

	// As a convenience when working with AZs generically, we expose
	// the AZ suffix alone, without the region name.
	// This can be used e.g. to create lookup tables by AZ letter that
	// work regardless of region.
	nameSuffix := (*az.ZoneName)[len(*az.RegionName):]

	d.SetId(aws.StringValue(az.ZoneName))
	d.Set("name", az.ZoneName)
	d.Set("name_suffix", nameSuffix)
	d.Set("region", az.RegionName)
	d.Set("state", az.State)
	d.Set("zone_id", az.ZoneId)

	return nil
}
