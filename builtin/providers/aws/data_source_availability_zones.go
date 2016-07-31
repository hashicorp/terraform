package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAvailabilityZones() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAvailabilityZonesRead,

		Schema: map[string]*schema.Schema{
			"names": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsAvailabilityZonesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading availability zones")
	d.SetId(time.Now().UTC().String())

	req := &ec2.DescribeAvailabilityZonesInput{DryRun: aws.Bool(false)}
	azresp, err := conn.DescribeAvailabilityZones(req)
	if err != nil {
		return fmt.Errorf("Error listing availability zones: %s", err)
	}

	raw := make([]string, len(azresp.AvailabilityZones))
	for i, v := range azresp.AvailabilityZones {
		raw[i] = *v.ZoneName
	}

	sort.Strings(raw)

	if err := d.Set("names", raw); err != nil {
		return fmt.Errorf("[WARN] Error setting availability zones")
	}

	return nil
}
