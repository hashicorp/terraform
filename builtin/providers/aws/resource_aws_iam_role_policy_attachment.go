package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamRolePolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamRolePolicyAttachmentCreate,
		Read:   resourceAwsIamRolePolicyAttachmentRead,
		Delete: resourceAwsIamRolePolicyAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamRolePolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	role := d.Get("role").(string)
	arn := d.Get("policy_arn").(string)

	err := attachPolicyToRole(conn, role, arn)
	if err != nil {
		return fmt.Errorf("[WARN] Error attaching policy %s to IAM Role %s: %v", arn, role, err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", role)))
	return resourceAwsIamRolePolicyAttachmentRead(d, meta)
}

func resourceAwsIamRolePolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	role := d.Get("role").(string)
	arn := d.Get("policy_arn").(string)

	_, err := conn.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(role),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] No such entity found for Policy Attachment (%s)", role)
				d.SetId("")
				return nil
			}
		}
		return err
	}

	args := iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(role),
	}
	var policy string
	err = conn.ListAttachedRolePoliciesPages(&args, func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
		for _, p := range page.AttachedPolicies {
			if *p.PolicyArn == arn {
				policy = *p.PolicyArn
			}
		}

		return policy == ""
	})
	if err != nil {
		return err
	}
	if policy == "" {
		log.Printf("[WARN] No such policy found for Role Policy Attachment (%s)", role)
		d.SetId("")
	}

	return nil
}

func resourceAwsIamRolePolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	role := d.Get("role").(string)
	arn := d.Get("policy_arn").(string)

	err := detachPolicyFromRole(conn, role, arn)
	if err != nil {
		return fmt.Errorf("[WARN] Error removing policy %s from IAM Role %s: %v", arn, role, err)
	}
	return nil
}

func attachPolicyToRole(conn *iam.IAM, role string, arn string) error {
	_, err := conn.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(role),
		PolicyArn: aws.String(arn),
	})
	if err != nil {
		return err
	}
	return nil
}

func detachPolicyFromRole(conn *iam.IAM, role string, arn string) error {
	_, err := conn.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  aws.String(role),
		PolicyArn: aws.String(arn),
	})
	if err != nil {
		return err
	}
	return nil
}
