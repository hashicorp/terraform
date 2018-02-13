package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpcEndpoint() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcEndpointRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"service_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_endpoint_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"route_table_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"prefix_list_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cidr_blocks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"network_interface_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"security_group_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"private_dns_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"dns_entry": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dns_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"hosted_zone_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsVpcEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeVpcEndpointsInput{}

	if id, ok := d.GetOk("id"); ok {
		req.VpcEndpointIds = aws.StringSlice([]string{id.(string)})
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"vpc-endpoint-state": d.Get("state").(string),
			"vpc-id":             d.Get("vpc_id").(string),
			"service-name":       d.Get("service_name").(string),
		},
	)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading VPC Endpoint: %s", req)
	resp, err := conn.DescribeVpcEndpoints(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.VpcEndpoints) == 0 {
		return fmt.Errorf("no matching VPC endpoint found")
	}
	if len(resp.VpcEndpoints) > 1 {
		return fmt.Errorf("multiple VPC endpoints matched; use additional constraints to reduce matches to a single VPC endpoint")
	}

	vpce := resp.VpcEndpoints[0]
	d.SetId(aws.StringValue(vpce.VpcEndpointId))

	return vpcEndpointAttributes(d, vpce, conn)
}
