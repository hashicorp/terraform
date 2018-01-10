package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsNatGateway() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsNatGatewayRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),

			"filter": ec2CustomFiltersSchema(),
		},
	}
}

func dataSourceAwsNatGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeNatGatewaysInput{}

	if id, ok := d.GetOk("id"); ok {
		req.NatGatewayIds = aws.StringSlice([]string{id.(string)})
	}

	if vpc_id, ok := d.GetOk("vpc_id"); ok {
		req.Filter = append(req.Filter, buildEC2AttributeFilterList(
			map[string]string{
				"vpc-id": vpc_id.(string),
			},
		)...)
	}

	if state, ok := d.GetOk("state"); ok {
		req.Filter = append(req.Filter, buildEC2AttributeFilterList(
			map[string]string{
				"state": state.(string),
			},
		)...)
	}

	if subnet_id, ok := d.GetOk("subnet_id"); ok {
		req.Filter = append(req.Filter, buildEC2AttributeFilterList(
			map[string]string{
				"subnet-id": subnet_id.(string),
			},
		)...)
	}

	req.Filter = append(req.Filter, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filter) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filter = nil
	}
	log.Printf("[DEBUG] Reading NAT Gateway: %s", req)
	resp, err := conn.DescribeNatGateways(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.NatGateways) == 0 {
		return fmt.Errorf("no matching NAT gateway found: %#v", req)
	}
	if len(resp.NatGateways) > 1 {
		return fmt.Errorf("multiple NAT gateways matched; use additional constraints to reduce matches to a single NAT gateway")
	}

	ngw := resp.NatGateways[0]

	d.SetId(aws.StringValue(ngw.NatGatewayId))
	d.Set("state", ngw.State)
	d.Set("subnet_id", ngw.SubnetId)
	d.Set("vpc_id", ngw.VpcId)

	for _, address := range ngw.NatGatewayAddresses {
		if *address.AllocationId != "" {
			d.Set("allocation_id", address.AllocationId)
			d.Set("network_interface_id", address.NetworkInterfaceId)
			d.Set("private_ip", address.PrivateIp)
			d.Set("public_ip", address.PublicIp)
			break
		}
	}

	return nil
}
