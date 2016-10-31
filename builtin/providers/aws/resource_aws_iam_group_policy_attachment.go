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

func resourceAwsIamGroupPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamGroupPolicyAttachmentCreate,
		Read:   resourceAwsIamGroupPolicyAttachmentRead,
		Delete: resourceAwsIamGroupPolicyAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"group": {
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

func resourceAwsIamGroupPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	group := d.Get("group").(string)
	arn := d.Get("policy_arn").(string)

	err := attachPolicyToGroup(conn, group, arn)
	if err != nil {
		return fmt.Errorf("[WARN] Error attaching policy %s to IAM group %s: %v", arn, group, err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", group)))
	return resourceAwsIamGroupPolicyAttachmentRead(d, meta)
}

func resourceAwsIamGroupPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	group := d.Get("group").(string)
	arn := d.Get("policy_arn").(string)

	_, err := conn.GetGroup(&iam.GetGroupInput{
		GroupName: aws.String(group),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] No such entity found for Policy Attachment (%s)", group)
				d.SetId("")
				return nil
			}
		}
		return err
	}

	attachedPolicies, err := conn.ListAttachedGroupPolicies(&iam.ListAttachedGroupPoliciesInput{
		GroupName: aws.String(group),
	})
	if err != nil {
		return err
	}

	var policy string
	for _, p := range attachedPolicies.AttachedPolicies {
		if *p.PolicyArn == arn {
			policy = *p.PolicyArn
		}
	}

	if policy == "" {
		log.Printf("[WARN] No such policy found for Group Policy Attachment (%s)", group)
		d.SetId("")
	}

	return nil
}

func resourceAwsIamGroupPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	group := d.Get("group").(string)
	arn := d.Get("policy_arn").(string)

	err := detachPolicyFromGroup(conn, group, arn)
	if err != nil {
		return fmt.Errorf("[WARN] Error removing policy %s from IAM Group %s: %v", arn, group, err)
	}
	return nil
}

func attachPolicyToGroup(conn *iam.IAM, group string, arn string) error {
	_, err := conn.AttachGroupPolicy(&iam.AttachGroupPolicyInput{
		GroupName: aws.String(group),
		PolicyArn: aws.String(arn),
	})
	if err != nil {
		return err
	}
	return nil
}

func detachPolicyFromGroup(conn *iam.IAM, group string, arn string) error {
	_, err := conn.DetachGroupPolicy(&iam.DetachGroupPolicyInput{
		GroupName: aws.String(group),
		PolicyArn: aws.String(arn),
	})
	if err != nil {
		return err
	}
	return nil
}
