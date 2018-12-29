package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSecurityGroupRead,

		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": ec2CustomFiltersSchema(),

			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeSecurityGroupsInput{}

	if id, ok := d.GetOk("id"); ok {
		req.GroupIds = []*string{aws.String(id.(string))}
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"group-name": d.Get("name").(string),
			"vpc-id":     d.Get("vpc_id").(string),
		},
	)
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading Security Group: %s", req)
	resp, err := conn.DescribeSecurityGroups(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.SecurityGroups) == 0 {
		return fmt.Errorf("no matching SecurityGroup found")
	}
	if len(resp.SecurityGroups) > 1 {
		return fmt.Errorf("multiple Security Groups matched; use additional constraints to reduce matches to a single Security Group")
	}

	sg := resp.SecurityGroups[0]

	d.SetId(*sg.GroupId)
	d.Set("name", sg.GroupName)
	d.Set("description", sg.Description)
	d.Set("vpc_id", sg.VpcId)
	d.Set("tags", tagsToMap(sg.Tags))
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "ec2",
		Region:    meta.(*AWSClient).region,
		AccountID: *sg.OwnerId,
		Resource:  fmt.Sprintf("security-group/%s", *sg.GroupId),
	}.String()
	d.Set("arn", arn)

	return nil
}
