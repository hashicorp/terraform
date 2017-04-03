package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsIAMRole() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIAMRoleRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"assume_role_policy_document": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAwsIAMRoleRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	roleName := d.Get("role_name").(string)

	req := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	resp, err := iamconn.GetRole(req)
	if err != nil {
		return errwrap.Wrapf("Error getting roles: {{err}}", err)
	}
	if resp == nil {
		return fmt.Errorf("no IAM role found")
	}

	role := resp.Role

	d.SetId(*role.RoleId)
	d.Set("arn", role.Arn)
	d.Set("assume_role_policy_document", role.AssumeRolePolicyDocument)
	d.Set("path", role.Path)
	d.Set("role_id", role.RoleId)

	return nil
}
