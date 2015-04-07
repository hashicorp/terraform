package aws

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamRolePolicy() *schema.Resource {
	return &schema.Resource{
		// PutRolePolicy API is idempotent, so these can be the same.
		Create: resourceAwsIamRolePolicyPut,
		Update: resourceAwsIamRolePolicyPut,

		Read:   resourceAwsIamRolePolicyRead,
		Delete: resourceAwsIamRolePolicyDelete,

		Schema: map[string]*schema.Schema{
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamRolePolicyPut(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.PutRolePolicyRequest{
		RoleName:       aws.String(d.Get("role").(string)),
		PolicyName:     aws.String(d.Get("name").(string)),
		PolicyDocument: aws.String(d.Get("policy").(string)),
	}

	if err := iamconn.PutRolePolicy(request); err != nil {
		return fmt.Errorf("Error putting IAM role policy %s: %s", request.PolicyName, err)
	}

	d.SetId(fmt.Sprintf("%s:%s", *request.RoleName, *request.PolicyName))
	return nil
}

func resourceAwsIamRolePolicyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	role, name := resourceAwsIamRolePolicyParseId(d)

	request := &iam.GetRolePolicyRequest{
		PolicyName: aws.String(name),
		RoleName:   aws.String(role),
	}

	getResp, err := iamconn.GetRolePolicy(request)
	if err != nil {
		if iamerr, ok := err.(aws.APIError); ok && iamerr.Code == "NoSuchEntity" { // XXX test me
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM policy %s from role %s: %s", name, role, err)
	}

	if getResp.PolicyDocument == nil {
		return fmt.Errorf("GetRolePolicy returned a nil policy document")
	}

	policy, err := url.QueryUnescape(*getResp.PolicyDocument)
	if err != nil {
		return err
	}
	return d.Set("policy", policy)
}

func resourceAwsIamRolePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	role, name := resourceAwsIamRolePolicyParseId(d)

	request := &iam.DeleteRolePolicyRequest{
		PolicyName: aws.String(name),
		RoleName:   aws.String(role),
	}

	if err := iamconn.DeleteRolePolicy(request); err != nil {
		return fmt.Errorf("Error deleting IAM role policy %s: %s", d.Id(), err)
	}
	return nil
}

func resourceAwsIamRolePolicyParseId(d *schema.ResourceData) (userName, policyName string) {
	parts := strings.SplitN(d.Id(), ":", 2)
	userName = parts[0]
	policyName = parts[1]
	return
}
