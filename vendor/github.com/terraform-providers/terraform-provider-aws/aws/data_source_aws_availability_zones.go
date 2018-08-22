package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func dataSourceAwsAvailabilityZones() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAvailabilityZonesRead,

		Schema: map[string]*schema.Schema{
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.AvailabilityZoneStateAvailable,
					ec2.AvailabilityZoneStateInformation,
					ec2.AvailabilityZoneStateImpaired,
					ec2.AvailabilityZoneStateUnavailable,
				}, false),
			},
		},
	}
}

func dataSourceAwsAvailabilityZonesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading Availability Zones.")
	d.SetId(time.Now().UTC().String())

	request := &ec2.DescribeAvailabilityZonesInput{}

	if v, ok := d.GetOk("state"); ok {
		request.Filters = []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String(v.(string))},
			},
		}
	}

	log.Printf("[DEBUG] Reading Availability Zones: %s", request)
	resp, err := conn.DescribeAvailabilityZones(request)
	if err != nil {
		return fmt.Errorf("Error fetching Availability Zones: %s", err)
	}

	raw := make([]string, len(resp.AvailabilityZones))
	for i, v := range resp.AvailabilityZones {
		raw[i] = *v.ZoneName
	}

	sort.Strings(raw)

	if err := d.Set("names", raw); err != nil {
		return fmt.Errorf("[WARN] Error setting Availability Zones: %s", err)
	}

	return nil
}
