package aws

import (
	"fmt"
	"log"
	"strconv"

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
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"service_name"},
			},
			"service_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"service"},
			},
			"service_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_endpoint_policy_supported": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"acceptance_required": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"availability_zones": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},
			"private_dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"base_endpoint_dns_names": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceAwsVpcEndpointServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	var serviceName string
	if v, ok := d.GetOk("service_name"); ok {
		serviceName = v.(string)
	} else if v, ok := d.GetOk("service"); ok {
		serviceName = fmt.Sprintf("com.amazonaws.%s.%s", meta.(*AWSClient).region, v.(string))
	} else {
		return fmt.Errorf(
			"One of ['service', 'service_name'] must be set to query VPC Endpoint Services")
	}

	req := &ec2.DescribeVpcEndpointServicesInput{
		ServiceNames: aws.StringSlice([]string{serviceName}),
	}

	log.Printf("[DEBUG] Reading VPC Endpoint Services: %s", req)
	resp, err := conn.DescribeVpcEndpointServices(req)
	if err != nil {
		return fmt.Errorf("Error fetching VPC Endpoint Services: %s", err)
	}

	if resp == nil || len(resp.ServiceNames) == 0 {
		return fmt.Errorf("no matching VPC Endpoint Service found")
	}

	if len(resp.ServiceNames) > 1 {
		return fmt.Errorf("multiple VPC Endpoint Services matched; use additional constraints to reduce matches to a single VPC Endpoint Service")
	}

	// Note: AWS Commercial now returns a response with `ServiceNames` and
	// `ServiceDetails`, but GovCloud responses only include `ServiceNames`
	if len(resp.ServiceDetails) == 0 {
		d.SetId(strconv.Itoa(hashcode.String(*resp.ServiceNames[0])))
		d.Set("service_name", resp.ServiceNames[0])
		return nil
	}

	sd := resp.ServiceDetails[0]
	serviceName = aws.StringValue(sd.ServiceName)
	d.SetId(strconv.Itoa(hashcode.String(serviceName)))
	d.Set("service_name", serviceName)
	d.Set("service_type", sd.ServiceType[0].ServiceType)
	d.Set("owner", sd.Owner)
	d.Set("vpc_endpoint_policy_supported", sd.VpcEndpointPolicySupported)
	d.Set("acceptance_required", sd.AcceptanceRequired)
	d.Set("availability_zones", flattenStringList(sd.AvailabilityZones))
	d.Set("private_dns_name", sd.PrivateDnsName)
	d.Set("base_endpoint_dns_names", flattenStringList(sd.BaseEndpointDnsNames))

	return nil
}
