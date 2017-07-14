package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
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
		},
	}
}

func dataSourceAwsSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeSecurityGroupsInput{}

	if id, idExists := d.GetOk("id"); idExists {
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

	log.Printf("[DEBUG] Describe Security Groups %v\n", req)
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
	d.Set("id", sg.VpcId)
	d.Set("name", sg.GroupName)
	d.Set("description", sg.Description)
	d.Set("vpc_id", sg.VpcId)
	d.Set("tags", tagsToMap(sg.Tags))
	d.Set("arn", fmt.Sprintf("arn:%s:ec2:%s:%s/security-group/%s",
		meta.(*AWSClient).partition, meta.(*AWSClient).region, *sg.OwnerId, *sg.GroupId))

	return nil
}
