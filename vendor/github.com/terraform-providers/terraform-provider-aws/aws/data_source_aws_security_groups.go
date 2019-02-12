package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSecurityGroups() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSecurityGroupsRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),
			"tags":   tagsSchemaComputed(),

			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"vpc_ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsSecurityGroupsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeSecurityGroupsInput{}

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("tags")

	if !filtersOk && !tagsOk {
		return fmt.Errorf("One of filters or tags must be assigned")
	}

	if filtersOk {
		req.Filters = append(req.Filters,
			buildAwsDataSourceFilters(filters.(*schema.Set))...)
	}
	if tagsOk {
		req.Filters = append(req.Filters, buildEC2TagFilterList(
			tagsFromMap(tags.(map[string]interface{})),
		)...)
	}

	log.Printf("[DEBUG] Reading Security Groups with request: %s", req)

	var ids, vpc_ids []string
	for {
		resp, err := conn.DescribeSecurityGroups(req)
		if err != nil {
			return fmt.Errorf("error reading security groups: %s", err)
		}

		for _, sg := range resp.SecurityGroups {
			ids = append(ids, aws.StringValue(sg.GroupId))
			vpc_ids = append(vpc_ids, aws.StringValue(sg.VpcId))
		}

		if resp.NextToken == nil {
			break
		}
		req.NextToken = resp.NextToken
	}

	if len(ids) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	log.Printf("[DEBUG] Found %d security groups via given filter: %s", len(ids), req)

	d.SetId(resource.UniqueId())
	err := d.Set("ids", ids)
	if err != nil {
		return err
	}

	err = d.Set("vpc_ids", vpc_ids)
	return err
}
