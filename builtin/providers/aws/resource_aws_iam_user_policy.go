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

func resourceAwsIamUserPolicy() *schema.Resource {
	return &schema.Resource{
		// PutUserPolicy API is idempotent, so these can be the same.
		Create: resourceAwsIamUserPolicyPut,
		Update: resourceAwsIamUserPolicyPut,

		Read:   resourceAwsIamUserPolicyRead,
		Delete: resourceAwsIamUserPolicyDelete,

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
			"user": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamUserPolicyPut(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.PutUserPolicyInput{
		UserName:       aws.String(d.Get("user").(string)),
		PolicyName:     aws.String(d.Get("name").(string)),
		PolicyDocument: aws.String(d.Get("policy").(string)),
	}

	if _, err := iamconn.PutUserPolicy(request); err != nil {
		return fmt.Errorf("Error putting IAM user policy %s: %s", *request.PolicyName, err)
	}

	d.SetId(fmt.Sprintf("%s:%s", *request.UserName, *request.PolicyName))
	return nil
}

func resourceAwsIamUserPolicyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	user, name := resourceAwsIamUserPolicyParseId(d.Id())

	request := &iam.GetUserPolicyInput{
		PolicyName: aws.String(name),
		UserName:   aws.String(user),
	}

	var err error
	getResp, err := iamconn.GetUserPolicy(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX test me
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM policy %s from user %s: %s", name, user, err)
	}

	if getResp.PolicyDocument == nil {
		return fmt.Errorf("GetUserPolicy returned a nil policy document")
	}

	policy, err := url.QueryUnescape(*getResp.PolicyDocument)
	if err != nil {
		return err
	}
	return d.Set("policy", policy)
}

func resourceAwsIamUserPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	user, name := resourceAwsIamUserPolicyParseId(d.Id())

	request := &iam.DeleteUserPolicyInput{
		PolicyName: aws.String(name),
		UserName:   aws.String(user),
	}

	if _, err := iamconn.DeleteUserPolicy(request); err != nil {
		return fmt.Errorf("Error deleting IAM user policy %s: %s", d.Id(), err)
	}
	return nil
}

func resourceAwsIamUserPolicyParseId(id string) (userName, policyName string) {
	parts := strings.SplitN(id, ":", 2)
	userName = parts[0]
	policyName = parts[1]
	return
}
