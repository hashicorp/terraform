package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpcEndpointService() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcEndpointServiceRead,

		Schema: map[string]*schema.Schema{
			"service": {
				Type:     schema.TypeString,
				Required: true,
			},
			"service_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsVpcEndpointServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	service := d.Get("service").(string)
	request := &ec2.DescribeVpcEndpointServicesInput{}

	log.Printf("[DEBUG] Reading VPC Endpoint Service: %s", request)
	resp, err := conn.DescribeVpcEndpointServices(request)
	if err != nil {
		return fmt.Errorf("Error fetching VPC Endpoint Services: %s", err)
	}

	names := aws.StringValueSlice(resp.ServiceNames)
	for _, name := range names {
		if strings.HasSuffix(name, "."+service) {
			d.SetId(strconv.Itoa(hashcode.String(name)))
			d.Set("service_name", name)
			return nil
		}
	}

	return fmt.Errorf("VPC Endpoint Service (%s) not found", service)
}
