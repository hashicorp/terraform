package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSubnetIDs() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSubnetIDsRead,
		Schema: map[string]*schema.Schema{

			"tags": tagsSchemaComputed(),

			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceAwsSubnetIDsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSubnetsInput{}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"vpc-id": d.Get("vpc_id").(string),
		},
	)

	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)

	log.Printf("[DEBUG] DescribeSubnets %s\n", req)
	resp, err := conn.DescribeSubnets(req)
	if err != nil {
		return err
	}

	if resp == nil || len(resp.Subnets) == 0 {
		return fmt.Errorf("no matching subnet found for vpc with id %s", d.Get("vpc_id").(string))
	}

	subnets := make([]string, 0)

	for _, subnet := range resp.Subnets {
		subnets = append(subnets, *subnet.SubnetId)
	}

	d.SetId(d.Get("vpc_id").(string))
	d.Set("ids", subnets)

	return nil
}
