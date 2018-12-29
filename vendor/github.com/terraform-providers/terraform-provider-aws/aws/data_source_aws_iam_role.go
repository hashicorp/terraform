package aws

import (
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
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
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use `assume_role_policy` instead",
			},
			"assume_role_policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"permissions_boundary": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role_id": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use `unique_id` instead",
			},
			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role_name": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use `name` instead",
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"max_session_duration": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsIAMRoleRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	name, hasName := d.GetOk("name")
	roleName, hasRoleName := d.GetOk("role_name")

	if !hasName && !hasRoleName {
		return fmt.Errorf("`%s` must be set", "name")
	}

	var id string
	if hasName {
		id = name.(string)
	} else if hasRoleName {
		id = roleName.(string)
	}
	d.SetId(id)

	input := &iam.GetRoleInput{
		RoleName: aws.String(d.Id()),
	}

	output, err := iamconn.GetRole(input)
	if err != nil {
		return fmt.Errorf("Error reading IAM Role %s: %s", d.Id(), err)
	}

	d.Set("arn", output.Role.Arn)
	if err := d.Set("create_date", output.Role.CreateDate.Format(time.RFC3339)); err != nil {
		return err
	}
	d.Set("description", output.Role.Description)
	d.Set("max_session_duration", output.Role.MaxSessionDuration)
	d.Set("name", output.Role.RoleName)
	d.Set("path", output.Role.Path)
	d.Set("permissions_boundary", "")
	if output.Role.PermissionsBoundary != nil {
		d.Set("permissions_boundary", output.Role.PermissionsBoundary.PermissionsBoundaryArn)
	}
	d.Set("unique_id", output.Role.RoleId)

	assumRolePolicy, err := url.QueryUnescape(aws.StringValue(output.Role.AssumeRolePolicyDocument))
	if err != nil {
		return err
	}
	if err := d.Set("assume_role_policy", assumRolePolicy); err != nil {
		return err
	}

	// Keep backward compatibility with previous attributes
	d.Set("role_id", output.Role.RoleId)
	d.Set("assume_role_policy_document", assumRolePolicy)

	return nil
}
