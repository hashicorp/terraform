package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamRoleCreate,
		Read:   resourceAwsIamRoleRead,
		// TODO
		//Update: resourceAwsIamRoleUpdate,
		Delete: resourceAwsIamRoleDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},
			"assume_role_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamRoleCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

	request := &iam.CreateRoleInput{
		Path:                     aws.String(d.Get("path").(string)),
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(d.Get("assume_role_policy").(string)),
	}

	createResp, err := iamconn.CreateRole(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM Role %s: %s", name, err)
	}
	return resourceAwsIamRoleReadResult(d, createResp.Role)
}

func resourceAwsIamRoleRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.GetRoleInput{
		RoleName: aws.String(d.Id()),
	}

	getResp, err := iamconn.GetRole(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX test me
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM Role %s: %s", d.Id(), err)
	}
	return resourceAwsIamRoleReadResult(d, getResp.Role)
}

func resourceAwsIamRoleReadResult(d *schema.ResourceData, role *iam.Role) error {
	d.SetId(*role.RoleName)
	if err := d.Set("name", role.RoleName); err != nil {
		return err
	}
	if err := d.Set("arn", role.ARN); err != nil {
		return err
	}
	if err := d.Set("path", role.Path); err != nil {
		return err
	}
	if err := d.Set("unique_id", role.RoleID); err != nil {
		return err
	}
	return nil
}

func resourceAwsIamRoleDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.DeleteRoleInput{
		RoleName: aws.String(d.Id()),
	}

	if _, err := iamconn.DeleteRole(request); err != nil {
		return fmt.Errorf("Error deleting IAM Role %s: %s", d.Id(), err)
	}
	return nil
}
