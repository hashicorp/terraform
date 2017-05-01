package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPlacementGroupImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).ec2conn

	id := d.Id()
	resp, err := conn.DescribePlacementGroups(&ec2.DescribePlacementGroupsInput{
		GroupNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.PlacementGroups) < 1 || resp.PlacementGroups[0] == nil {
		return nil, fmt.Errorf("Placement Group %s is not found", id)
	}
	pg := resp.PlacementGroups[0]

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	d.SetId(id)
	d.SetType("aws_placement_group")
	d.Set("name", pg.GroupName)
	d.Set("strategy", pg.Strategy)

	return results, nil

}
