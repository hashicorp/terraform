package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsIAMGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIAMGroupRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"group_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAwsIAMGroupRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	groupName := d.Get("group_name").(string)

	req := &iam.GetGroupInput{
		GroupName: aws.String(groupName),
	}

	log.Printf("[DEBUG] Reading IAM Group: %s", req)
	resp, err := iamconn.GetGroup(req)
	if err != nil {
		return errwrap.Wrapf("Error getting group: {{err}}", err)
	}
	if resp == nil {
		return fmt.Errorf("no IAM group found")
	}

	group := resp.Group

	d.SetId(*group.GroupId)
	d.Set("arn", group.Arn)
	d.Set("path", group.Path)
	d.Set("group_id", group.GroupId)

	return nil
}
