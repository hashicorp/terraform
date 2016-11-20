package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpcEndpointServices() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcEndpointServicesRead,

		Schema: map[string]*schema.Schema{
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsVpcEndpointServicesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading VPC Endpoint Services.")
	d.SetId(time.Now().UTC().String())

	request := &ec2.DescribeVpcEndpointServicesInput{}

	resp, err := conn.DescribeVpcEndpointServices(request)
	if err != nil {
		return fmt.Errorf("Error fetching VPC Endpoint Services: %s", err)
	}

	services := aws.StringValueSlice(resp.ServiceNames)

	if err := d.Set("names", services); err != nil {
		return fmt.Errorf("[WARN] Error setting VPC Endpoint Services: %s", err)
	}

	return nil
}
