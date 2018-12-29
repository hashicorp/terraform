package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsIamPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamPolicyAttachmentCreate,
		Read:   resourceAwsIamPolicyAttachmentRead,
		Update: resourceAwsIamPolicyAttachmentUpdate,
		Delete: resourceAwsIamPolicyAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"users": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"policy_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	name := d.Get("name").(string)
	arn := d.Get("policy_arn").(string)
	users := expandStringList(d.Get("users").(*schema.Set).List())
	roles := expandStringList(d.Get("roles").(*schema.Set).List())
	groups := expandStringList(d.Get("groups").(*schema.Set).List())

	if len(users) == 0 && len(roles) == 0 && len(groups) == 0 {
		return fmt.Errorf("No Users, Roles, or Groups specified for IAM Policy Attachment %s", name)
	} else {
		var userErr, roleErr, groupErr error
		if users != nil {
			userErr = attachPolicyToUsers(conn, users, arn)
		}
		if roles != nil {
			roleErr = attachPolicyToRoles(conn, roles, arn)
		}
		if groups != nil {
			groupErr = attachPolicyToGroups(conn, groups, arn)
		}
		if userErr != nil || roleErr != nil || groupErr != nil {
			return composeErrors(fmt.Sprint("[WARN] Error attaching policy with IAM Policy Attachment ", name, ":"), userErr, roleErr, groupErr)
		}
	}
	d.SetId(d.Get("name").(string))
	return resourceAwsIamPolicyAttachmentRead(d, meta)
}

func resourceAwsIamPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	arn := d.Get("policy_arn").(string)
	name := d.Get("name").(string)

	_, err := conn.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: aws.String(arn),
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

	ul := make([]string, 0)
	rl := make([]string, 0)
	gl := make([]string, 0)

	args := iam.ListEntitiesForPolicyInput{
		PolicyArn: aws.String(arn),
	}
	err = conn.ListEntitiesForPolicyPages(&args, func(page *iam.ListEntitiesForPolicyOutput, lastPage bool) bool {
		for _, u := range page.PolicyUsers {
			ul = append(ul, *u.UserName)
		}

		for _, r := range page.PolicyRoles {
			rl = append(rl, *r.RoleName)
		}

		for _, g := range page.PolicyGroups {
			gl = append(gl, *g.GroupName)
		}
		return true
	})
	if err != nil {
		return err
	}

	userErr := d.Set("users", ul)
	roleErr := d.Set("roles", rl)
	groupErr := d.Set("groups", gl)

	if userErr != nil || roleErr != nil || groupErr != nil {
		return composeErrors(fmt.Sprint("[WARN} Error setting user, role, or group list from IAM Policy Attachment ", name, ":"), userErr, roleErr, groupErr)
	}

	return nil
}
func resourceAwsIamPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	var userErr, roleErr, groupErr error

	if d.HasChange("users") {
		userErr = updateUsers(conn, d, meta)
	}
	if d.HasChange("roles") {
		roleErr = updateRoles(conn, d, meta)
	}
	if d.HasChange("groups") {
		groupErr = updateGroups(conn, d, meta)
	}
	if userErr != nil || roleErr != nil || groupErr != nil {
		return composeErrors(fmt.Sprint("[WARN] Error updating user, role, or group list from IAM Policy Attachment ", name, ":"), userErr, roleErr, groupErr)
	}
	return resourceAwsIamPolicyAttachmentRead(d, meta)
}

func resourceAwsIamPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	arn := d.Get("policy_arn").(string)
	users := expandStringList(d.Get("users").(*schema.Set).List())
	roles := expandStringList(d.Get("roles").(*schema.Set).List())
	groups := expandStringList(d.Get("groups").(*schema.Set).List())

	var userErr, roleErr, groupErr error
	if len(users) != 0 {
		userErr = detachPolicyFromUsers(conn, users, arn)
	}
	if len(roles) != 0 {
		roleErr = detachPolicyFromRoles(conn, roles, arn)
	}
	if len(groups) != 0 {
		groupErr = detachPolicyFromGroups(conn, groups, arn)
	}
	if userErr != nil || roleErr != nil || groupErr != nil {
		return composeErrors(fmt.Sprint("[WARN] Error removing user, role, or group list from IAM Policy Detach ", name, ":"), userErr, roleErr, groupErr)
	}
	return nil
}

func composeErrors(desc string, uErr error, rErr error, gErr error) error {
	errMsg := fmt.Sprintf(desc)
	errs := []error{uErr, rErr, gErr}
	for _, e := range errs {
		if e != nil {
			errMsg = errMsg + "\n– " + e.Error()
		}
	}
	return fmt.Errorf(errMsg)
}

func attachPolicyToUsers(conn *iam.IAM, users []*string, arn string) error {
	for _, u := range users {
		_, err := conn.AttachUserPolicy(&iam.AttachUserPolicyInput{
			UserName:  u,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func attachPolicyToRoles(conn *iam.IAM, roles []*string, arn string) error {
	for _, r := range roles {
		_, err := conn.AttachRolePolicy(&iam.AttachRolePolicyInput{
			RoleName:  r,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return err
		}

		var attachmentErr error
		attachmentErr = resource.Retry(2*time.Minute, func() *resource.RetryError {

			input := iam.ListRolePoliciesInput{
				RoleName: r,
			}

			attachedPolicies, err := conn.ListRolePolicies(&input)
			if err != nil {
				return resource.NonRetryableError(err)
			}

			if len(attachedPolicies.PolicyNames) > 0 {
				var foundPolicy bool
				for _, policyName := range attachedPolicies.PolicyNames {
					if strings.HasSuffix(arn, *policyName) {
						foundPolicy = true
						break
					}
				}

				if !foundPolicy {
					return resource.NonRetryableError(err)
				}
			}

			return nil
		})

		if attachmentErr != nil {
			return attachmentErr
		}
	}
	return nil
}
func attachPolicyToGroups(conn *iam.IAM, groups []*string, arn string) error {
	for _, g := range groups {
		_, err := conn.AttachGroupPolicy(&iam.AttachGroupPolicyInput{
			GroupName: g,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func updateUsers(conn *iam.IAM, d *schema.ResourceData, meta interface{}) error {
	arn := d.Get("policy_arn").(string)
	o, n := d.GetChange("users")
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

	if rErr := detachPolicyFromUsers(conn, remove, arn); rErr != nil {
		return rErr
	}
	if aErr := attachPolicyToUsers(conn, add, arn); aErr != nil {
		return aErr
	}
	return nil
}
func updateRoles(conn *iam.IAM, d *schema.ResourceData, meta interface{}) error {
	arn := d.Get("policy_arn").(string)
	o, n := d.GetChange("roles")
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

	if rErr := detachPolicyFromRoles(conn, remove, arn); rErr != nil {
		return rErr
	}
	if aErr := attachPolicyToRoles(conn, add, arn); aErr != nil {
		return aErr
	}
	return nil
}
func updateGroups(conn *iam.IAM, d *schema.ResourceData, meta interface{}) error {
	arn := d.Get("policy_arn").(string)
	o, n := d.GetChange("groups")
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

	if rErr := detachPolicyFromGroups(conn, remove, arn); rErr != nil {
		return rErr
	}
	if aErr := attachPolicyToGroups(conn, add, arn); aErr != nil {
		return aErr
	}
	return nil

}
func detachPolicyFromUsers(conn *iam.IAM, users []*string, arn string) error {
	for _, u := range users {
		_, err := conn.DetachUserPolicy(&iam.DetachUserPolicyInput{
			UserName:  u,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func detachPolicyFromRoles(conn *iam.IAM, roles []*string, arn string) error {
	for _, r := range roles {
		_, err := conn.DetachRolePolicy(&iam.DetachRolePolicyInput{
			RoleName:  r,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func detachPolicyFromGroups(conn *iam.IAM, groups []*string, arn string) error {
	for _, g := range groups {
		_, err := conn.DetachGroupPolicy(&iam.DetachGroupPolicyInput{
			GroupName: g,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
