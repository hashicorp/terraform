package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamGroupPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamGroupPolicyAttachmentCreate,
		Read:   resourceAwsIamGroupPolicyAttachmentRead,
		Update: resourceAwsIamGroupPolicyAttachmentUpdate,
		Delete: resourceAwsIamGroupPolicyAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"group": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"policy_arns": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsIamGroupPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	name := d.Get("name").(string)
	group := d.Get("group").(string)
	arns := expandStringList(d.Get("policy_arns").(*schema.Set).List())

	if len(arns) == 0 {
		return fmt.Errorf("[WARN] No Policies specified for IAM Group Policy Attachment %s", name)
	}

	err := attachPoliciesToGroup(conn, group, arns)
	if err != nil {
		return fmt.Errorf("[WARN] Error attaching policy with IAM Group Policy Attachment %s: %v", name, err)
	}

	d.SetId(d.Get("name").(string))
	return resourceAwsIamGroupPolicyAttachmentRead(d, meta)
}

func resourceAwsIamGroupPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	group := d.Get("group").(string)

	_, err := conn.GetGroup(&iam.GetGroupInput{
		GroupName: aws.String(group),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] No such entity found for Policy Attachment (%s)", d.Id())
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

	policies := make([]string, 0, len(attachedPolicies.AttachedPolicies))

	for _, p := range attachedPolicies.AttachedPolicies {
		policies = append(policies, *p.PolicyArn)
	}

	err = d.Set("policy_arns", policies)
	if err != nil {
		return fmt.Errorf("[WARN} Error setting policy list from IAM Group Policy Attachment %s: %v", name, err)
	}

	return nil
}

func resourceAwsIamGroupPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

	if d.HasChange("policy_arns") {
		err := updateGroupPolicies(conn, d, meta)
		if err != nil {
			return fmt.Errorf("[WARN] Error updating policy list from IAM Group Policy Attachment %s: %v", name, err)
		}
	}
	return resourceAwsIamGroupPolicyAttachmentRead(d, meta)
}

func resourceAwsIamGroupPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	group := d.Get("group").(string)
	arns := expandStringList(d.Get("policy_arns").(*schema.Set).List())

	err := detachPoliciesFromGroup(conn, group, arns)
	if err != nil {
		return fmt.Errorf("[WARN] Error removing policies from group IAM Group Policy Detach %s: %v", name, err)
	}
	return nil
}

func attachPoliciesToGroup(conn *iam.IAM, group string, arns []*string) error {
	for _, a := range arns {
		_, err := conn.AttachGroupPolicy(&iam.AttachGroupPolicyInput{
			GroupName: aws.String(group),
			PolicyArn: a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func updateGroupPolicies(conn *iam.IAM, d *schema.ResourceData, meta interface{}) error {
	group := d.Get("group").(string)
	o, n := d.GetChange("policy_arns")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}
	os := o.(*schema.Set)
	ns := n.(*schema.Set)
	remove := expandStringList(os.Difference(ns).List())
	add := expandStringList(ns.Difference(os).List())

	if rErr := detachPoliciesFromGroup(conn, group, remove); rErr != nil {
		return rErr
	}
	if aErr := attachPoliciesToGroup(conn, group, add); aErr != nil {
		return aErr
	}
	return nil
}

func detachPoliciesFromGroup(conn *iam.IAM, group string, arns []*string) error {
	for _, a := range arns {
		_, err := conn.DetachGroupPolicy(&iam.DetachGroupPolicyInput{
			GroupName: aws.String(group),
			PolicyArn: a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
