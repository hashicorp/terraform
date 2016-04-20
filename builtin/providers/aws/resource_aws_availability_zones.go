package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAvailabilityZones() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAZCreate,
		Read:   resourceAwsAZRead,
		//Update: resourceAwsAZUpdate,
		//Delete: resourceAwsAZDelete,

		Schema: map[string]*schema.Schema{
			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsAZCreate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsAZRead(d, meta)
}

func resourceAwsAZRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeAvailabilityZonesInput{DryRun: aws.Bool(false)}
	azresp, err := conn.DescribeAvailabilityZones(req)
	if err != nil {
		return fmt.Errorf("Error listing availability zones: %s", err)
	}
	azl := make([]string, 0, len(azresp.AvailabilityZones))
	for _, v := range azresp.AvailabilityZones {
		azl = append(*v.ZoneName)
	}
	azErr := d.Set("availability_zones", azl)
	if azErr != nil {
		return fmt.Errorf("[WARN] Error setting availability zones")
	}
	return nil
}
