package aws

import (
	"fmt"

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

	data := resourceAwsIamRoleRead(d, meta)
	// Keep backward compatibility with previous attributes
	d.Set("role_id", d.Get("unique_id").(string))
	d.Set("assume_role_policy_document", d.Get("assume_role_policy").(string))

	return data
}
