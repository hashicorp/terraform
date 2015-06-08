package aws

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamGroupPolicy() *schema.Resource {
	return &schema.Resource{
		// PutGroupPolicy API is idempotent, so these can be the same.
		Create: resourceAwsIamGroupPolicyPut,
		Update: resourceAwsIamGroupPolicyPut,

		Read:   resourceAwsIamGroupPolicyRead,
		Delete: resourceAwsIamGroupPolicyDelete,

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
			"group": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamGroupPolicyPut(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.PutGroupPolicyInput{
		GroupName:      aws.String(d.Get("group").(string)),
		PolicyName:     aws.String(d.Get("name").(string)),
		PolicyDocument: aws.String(d.Get("policy").(string)),
	}

	if _, err := iamconn.PutGroupPolicy(request); err != nil {
		return fmt.Errorf("Error putting IAM group policy %s: %s", *request.PolicyName, err)
	}

	d.SetId(fmt.Sprintf("%s:%s", *request.GroupName, *request.PolicyName))
	return nil
}

func resourceAwsIamGroupPolicyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	group, name := resourceAwsIamGroupPolicyParseId(d.Id())

	request := &iam.GetGroupPolicyInput{
		PolicyName: aws.String(name),
		GroupName:  aws.String(group),
	}

	var err error
	getResp, err := iamconn.GetGroupPolicy(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX test me
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM policy %s from group %s: %s", name, group, err)
	}

	if getResp.PolicyDocument == nil {
		return fmt.Errorf("GetGroupPolicy returned a nil policy document")
	}

	policy, err := url.QueryUnescape(*getResp.PolicyDocument)
	if err != nil {
		return err
	}
	return d.Set("policy", policy)
}

func resourceAwsIamGroupPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	group, name := resourceAwsIamGroupPolicyParseId(d.Id())

	request := &iam.DeleteGroupPolicyInput{
		PolicyName: aws.String(name),
		GroupName:  aws.String(group),
	}

	if _, err := iamconn.DeleteGroupPolicy(request); err != nil {
		return fmt.Errorf("Error deleting IAM group policy %s: %s", d.Id(), err)
	}
	return nil
}

func resourceAwsIamGroupPolicyParseId(id string) (groupName, policyName string) {
	parts := strings.SplitN(id, ":", 2)
	groupName = parts[0]
	policyName = parts[1]
	return
}
