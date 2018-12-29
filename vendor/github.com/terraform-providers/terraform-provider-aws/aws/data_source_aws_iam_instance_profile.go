package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsIAMInstanceProfile() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIAMInstanceProfileRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsIAMInstanceProfileRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	name := d.Get("name").(string)

	req := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(name),
	}

	log.Printf("[DEBUG] Reading IAM Instance Profile: %s", req)
	resp, err := iamconn.GetInstanceProfile(req)
	if err != nil {
		return errwrap.Wrapf("Error getting instance profiles: {{err}}", err)
	}
	if resp == nil {
		return fmt.Errorf("no IAM instance profile found")
	}

	instanceProfile := resp.InstanceProfile

	d.SetId(*instanceProfile.InstanceProfileId)
	d.Set("arn", instanceProfile.Arn)
	d.Set("create_date", fmt.Sprintf("%v", instanceProfile.CreateDate))
	d.Set("path", instanceProfile.Path)

	for _, r := range instanceProfile.Roles {
		d.Set("role_id", r.RoleId)
	}

	return nil
}
