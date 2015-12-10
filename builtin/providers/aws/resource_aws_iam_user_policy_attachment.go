package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamUserPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserPolicyAttachmentCreate,
		Read:   resourceAwsIamUserPolicyAttachmentRead,
		Update: resourceAwsIamUserPolicyAttachmentUpdate,
		Delete: resourceAwsIamUserPolicyAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"user": &schema.Schema{
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

func resourceAwsIamUserPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	name := d.Get("name").(string)
	user := d.Get("user").(string)
	arns := expandStringList(d.Get("policy_arns").(*schema.Set).List())

	if len(arns) == 0 {
		return fmt.Errorf("[WARN] No Policies specified for IAM User Policy Attachment %s", name)
	}

	err := attachPoliciesToUser(conn, user, arns)
	if err != nil {
		return fmt.Errorf("[WARN] Error attaching policy with IAM User Policy Attachment %s: %v", name, err)
	}

	d.SetId(d.Get("name").(string))
	return resourceAwsIamUserPolicyAttachmentRead(d, meta)
}

func resourceAwsIamUserPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	user := d.Get("user").(string)

	_, err := conn.GetUser(&iam.GetUserInput{
		UserName: aws.String(user),
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

	attachedPolicies, err := conn.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(user),
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
		return fmt.Errorf("[WARN} Error setting policy list from IAM User Policy Attachment %s: %v", name, err)
	}

	return nil
}

func resourceAwsIamUserPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

	if d.HasChange("policy_arns") {
		err := updateUserPolicies(conn, d, meta)
		if err != nil {
			return fmt.Errorf("[WARN] Error updating policy list from IAM User Policy Attachment %s: %v", name, err)
		}
	}
	return resourceAwsIamUserPolicyAttachmentRead(d, meta)
}

func resourceAwsIamUserPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	user := d.Get("user").(string)
	arns := expandStringList(d.Get("policy_arns").(*schema.Set).List())

	err := detachPoliciesFromUser(conn, user, arns)
	if err != nil {
		return fmt.Errorf("[WARN] Error removing policies from user IAM User Policy Detach %s: %v", name, err)
	}
	return nil
}

func attachPoliciesToUser(conn *iam.IAM, user string, arns []*string) error {
	for _, a := range arns {
		_, err := conn.AttachUserPolicy(&iam.AttachUserPolicyInput{
			UserName:  aws.String(user),
			PolicyArn: a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func updateUserPolicies(conn *iam.IAM, d *schema.ResourceData, meta interface{}) error {
	user := d.Get("user").(string)
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

	if rErr := detachPoliciesFromUser(conn, user, remove); rErr != nil {
		return rErr
	}
	if aErr := attachPoliciesToUser(conn, user, add); aErr != nil {
		return aErr
	}
	return nil
}

func detachPoliciesFromUser(conn *iam.IAM, user string, arns []*string) error {
	for _, a := range arns {
		_, err := conn.DetachUserPolicy(&iam.DetachUserPolicyInput{
			UserName:  aws.String(user),
			PolicyArn: a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
