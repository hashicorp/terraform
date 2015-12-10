package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamRolePolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamRolePolicyAttachmentCreate,
		Read:   resourceAwsIamRolePolicyAttachmentRead,
		Update: resourceAwsIamRolePolicyAttachmentUpdate,
		Delete: resourceAwsIamRolePolicyAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": &schema.Schema{
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

func resourceAwsIamRolePolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	name := d.Get("name").(string)
	role := d.Get("role").(string)
	arns := expandStringList(d.Get("policy_arns").(*schema.Set).List())

	if len(arns) == 0 {
		return fmt.Errorf("[WARN] No Policies specified for IAM Role Policy Attachment %s", name)
	}

	err := attachPoliciesToRole(conn, role, arns)
	if err != nil {
		return fmt.Errorf("[WARN] Error attaching policy with IAM Role Policy Attachment %s: %v", name, err)
	}

	d.SetId(d.Get("name").(string))
	return resourceAwsIamRolePolicyAttachmentRead(d, meta)
}

func resourceAwsIamRolePolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	role := d.Get("role").(string)

	_, err := conn.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(role),
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

	attachedPolicies, err := conn.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(role),
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
		return fmt.Errorf("[WARN} Error setting policy list from IAM Role Policy Attachment %s: %v", name, err)
	}

	return nil
}

func resourceAwsIamRolePolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

	if d.HasChange("policy_arns") {
		err := updateRolePolicies(conn, d, meta)
		if err != nil {
			return fmt.Errorf("[WARN] Error updating policy list from IAM Role Policy Attachment %s: %v", name, err)
		}
	}
	return resourceAwsIamRolePolicyAttachmentRead(d, meta)
}

func resourceAwsIamRolePolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	role := d.Get("role").(string)
	arns := expandStringList(d.Get("policy_arns").(*schema.Set).List())

	err := detachPoliciesFromRole(conn, role, arns)
	if err != nil {
		return fmt.Errorf("[WARN] Error removing policies from role IAM Role Policy Detach %s: %v", name, err)
	}
	return nil
}

func attachPoliciesToRole(conn *iam.IAM, role string, arns []*string) error {
	for _, a := range arns {
		_, err := conn.AttachRolePolicy(&iam.AttachRolePolicyInput{
			RoleName:  aws.String(role),
			PolicyArn: a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func updateRolePolicies(conn *iam.IAM, d *schema.ResourceData, meta interface{}) error {
	role := d.Get("role").(string)
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

	if rErr := detachPoliciesFromRole(conn, role, remove); rErr != nil {
		return rErr
	}
	if aErr := attachPoliciesToRole(conn, role, add); aErr != nil {
		return aErr
	}
	return nil
}

func detachPoliciesFromRole(conn *iam.IAM, role string, arns []*string) error {
	for _, a := range arns {
		_, err := conn.DetachRolePolicy(&iam.DetachRolePolicyInput{
			RoleName:  aws.String(role),
			PolicyArn: a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
