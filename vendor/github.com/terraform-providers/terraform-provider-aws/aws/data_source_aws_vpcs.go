package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpcs() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcsRead,
		Schema: map[string]*schema.Schema{
			"filter": ec2CustomFiltersSchema(),

			"tags": tagsSchemaComputed(),

			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceAwsVpcsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("tags")

	req := &ec2.DescribeVpcsInput{}

	if tagsOk {
		req.Filters = buildEC2TagFilterList(
			tagsFromMap(tags.(map[string]interface{})),
		)
	}

	if filtersOk {
		req.Filters = append(req.Filters, buildEC2CustomFilterList(
			filters.(*schema.Set),
		)...)
	}
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] DescribeVpcs %s\n", req)
	resp, err := conn.DescribeVpcs(req)
	if err != nil {
		return err
	}

	if resp == nil || len(resp.Vpcs) == 0 {
		return fmt.Errorf("no matching VPC found")
	}

	vpcs := make([]string, 0)

	for _, vpc := range resp.Vpcs {
		vpcs = append(vpcs, aws.StringValue(vpc.VpcId))
	}

	d.SetId(time.Now().UTC().String())
	if err := d.Set("ids", vpcs); err != nil {
		return fmt.Errorf("Error setting vpc ids: %s", err)
	}

	return nil
}
