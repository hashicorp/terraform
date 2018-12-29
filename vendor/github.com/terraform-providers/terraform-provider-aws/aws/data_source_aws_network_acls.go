package aws

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsNetworkAcls() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsNetworkAclsRead,
		Schema: map[string]*schema.Schema{
			"filter": ec2CustomFiltersSchema(),

			"tags": tagsSchemaComputed(),

			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
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

func dataSourceAwsNetworkAclsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeNetworkAclsInput{}

	if v, ok := d.GetOk("vpc_id"); ok {
		req.Filters = buildEC2AttributeFilterList(
			map[string]string{
				"vpc-id": v.(string),
			},
		)
	}

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("tags")

	if tagsOk {
		req.Filters = append(req.Filters, buildEC2TagFilterList(
			tagsFromMap(tags.(map[string]interface{})),
		)...)
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

	log.Printf("[DEBUG] DescribeNetworkAcls %s\n", req)
	resp, err := conn.DescribeNetworkAcls(req)
	if err != nil {
		return err
	}

	if resp == nil || len(resp.NetworkAcls) == 0 {
		return errors.New("no matching network ACLs found")
	}

	networkAcls := make([]string, 0)

	for _, networkAcl := range resp.NetworkAcls {
		networkAcls = append(networkAcls, aws.StringValue(networkAcl.NetworkAclId))
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("ids", networkAcls); err != nil {
		return fmt.Errorf("Error setting network ACL ids: %s", err)
	}

	return nil
}
