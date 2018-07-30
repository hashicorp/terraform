package aws

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsIamRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamRoleCreate,
		Read:   resourceAwsIamRoleRead,
		Update: resourceAwsIamRoleUpdate,
		Delete: resourceAwsIamRoleDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsIamRoleImport,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/2485f5c/botocore/data/iam/2010-05-08/service-2.json#L8329-L8334
					value := v.(string)
					if len(value) > 64 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 64 characters", k))
					}
					if !regexp.MustCompile("^[\\w+=,.@-]*$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must match [\\w+=,.@-]", k))
					}
					return
				},
			},

			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/2485f5c/botocore/data/iam/2010-05-08/service-2.json#L8329-L8334
					value := v.(string)
					if len(value) > 32 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 32 characters, name is limited to 64", k))
					}
					if !regexp.MustCompile("^[\\w+=,.@-]*$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must match [\\w+=,.@-]", k))
					}
					return
				},
			},

			"path": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},

			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateIamRoleDescription,
			},

			"assume_role_policy": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
				ValidateFunc:     validateJsonString,
			},

			"force_detach_policies": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"max_session_duration": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3600,
				ValidateFunc: validation.IntBetween(3600, 43200),
			},
		},
	}
}

func resourceAwsIamRoleImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("force_detach_policies", false)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsIamRoleCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.UniqueId()
	}

	request := &iam.CreateRoleInput{
		Path:                     aws.String(d.Get("path").(string)),
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(d.Get("assume_role_policy").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		request.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("max_session_duration"); ok {
		request.MaxSessionDuration = aws.Int64(int64(v.(int)))
	}

	var createResp *iam.CreateRoleOutput
	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error
		createResp, err = iamconn.CreateRole(request)
		// IAM users (referenced in Principal field of assume policy)
		// can take ~30 seconds to propagate in AWS
		if isAWSErr(err, "MalformedPolicyDocument", "Invalid principal in policy") {
			return resource.RetryableError(err)
		}
		return resource.NonRetryableError(err)
	})
	if err != nil {
		return fmt.Errorf("Error creating IAM Role %s: %s", name, err)
	}
	d.SetId(*createResp.Role.RoleName)
	return resourceAwsIamRoleRead(d, meta)
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

	role := getResp.Role

	if err := d.Set("name", role.RoleName); err != nil {
		return err
	}
	if err := d.Set("max_session_duration", role.MaxSessionDuration); err != nil {
		return err
	}
	if err := d.Set("arn", role.Arn); err != nil {
		return err
	}
	if err := d.Set("path", role.Path); err != nil {
		return err
	}
	if err := d.Set("unique_id", role.RoleId); err != nil {
		return err
	}
	if err := d.Set("create_date", role.CreateDate.Format(time.RFC3339)); err != nil {
		return err
	}

	if role.Description != nil {
		// the description isn't present in the response to CreateRole.
		if err := d.Set("description", role.Description); err != nil {
			return err
		}
	}

	assumRolePolicy, err := url.QueryUnescape(*role.AssumeRolePolicyDocument)
	if err != nil {
		return err
	}
	if err := d.Set("assume_role_policy", assumRolePolicy); err != nil {
		return err
	}
	return nil
}

func resourceAwsIamRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if d.HasChange("assume_role_policy") {
		assumeRolePolicyInput := &iam.UpdateAssumeRolePolicyInput{
			RoleName:       aws.String(d.Id()),
			PolicyDocument: aws.String(d.Get("assume_role_policy").(string)),
		}
		_, err := iamconn.UpdateAssumeRolePolicy(assumeRolePolicyInput)
		if err != nil {
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error Updating IAM Role (%s) Assume Role Policy: %s", d.Id(), err)
		}
	}

	if d.HasChange("description") {
		roleDescriptionInput := &iam.UpdateRoleDescriptionInput{
			RoleName:    aws.String(d.Id()),
			Description: aws.String(d.Get("description").(string)),
		}
		_, err := iamconn.UpdateRoleDescription(roleDescriptionInput)
		if err != nil {
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error Updating IAM Role (%s) Assume Role Policy: %s", d.Id(), err)
		}
	}

	if d.HasChange("max_session_duration") {
		roleMaxDurationInput := &iam.UpdateRoleInput{
			RoleName:           aws.String(d.Id()),
			MaxSessionDuration: aws.Int64(int64(d.Get("max_session_duration").(int))),
		}
		_, err := iamconn.UpdateRole(roleMaxDurationInput)
		if err != nil {
			if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error Updating IAM Role (%s) Max Session Duration: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceAwsIamRoleDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	// Roles cannot be destroyed when attached to an existing Instance Profile
	resp, err := iamconn.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error listing Profiles for IAM Role (%s) when trying to delete: %s", d.Id(), err)
	}

	// Loop and remove this Role from any Profiles
	if len(resp.InstanceProfiles) > 0 {
		for _, i := range resp.InstanceProfiles {
			_, err := iamconn.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: i.InstanceProfileName,
				RoleName:            aws.String(d.Id()),
			})
			if err != nil {
				return fmt.Errorf("Error deleting IAM Role %s: %s", d.Id(), err)
			}
		}
	}

	if d.Get("force_detach_policies").(bool) {
		// For managed policies
		managedPolicies := make([]*string, 0)
		err = iamconn.ListAttachedRolePoliciesPages(&iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(d.Id()),
		}, func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
			for _, v := range page.AttachedPolicies {
				managedPolicies = append(managedPolicies, v.PolicyArn)
			}
			return len(page.AttachedPolicies) > 0
		})
		if err != nil {
			return fmt.Errorf("Error listing Policies for IAM Role (%s) when trying to delete: %s", d.Id(), err)
		}
		if len(managedPolicies) > 0 {
			for _, parn := range managedPolicies {
				_, err = iamconn.DetachRolePolicy(&iam.DetachRolePolicyInput{
					PolicyArn: parn,
					RoleName:  aws.String(d.Id()),
				})
				if err != nil {
					if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
						log.Printf("[WARN] Role policy attachment (%s) was already removed from role (%s)", aws.StringValue(parn), d.Id())
					} else {
						return fmt.Errorf("Error deleting IAM Role %s: %s", d.Id(), err)
					}
				}
			}
		}

		// For inline policies
		inlinePolicies := make([]*string, 0)
		err = iamconn.ListRolePoliciesPages(&iam.ListRolePoliciesInput{
			RoleName: aws.String(d.Id()),
		}, func(page *iam.ListRolePoliciesOutput, lastPage bool) bool {
			for _, v := range page.PolicyNames {
				inlinePolicies = append(inlinePolicies, v)
			}
			return len(page.PolicyNames) > 0
		})
		if err != nil {
			return fmt.Errorf("Error listing inline Policies for IAM Role (%s) when trying to delete: %s", d.Id(), err)
		}
		if len(inlinePolicies) > 0 {
			for _, pname := range inlinePolicies {
				_, err := iamconn.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
					PolicyName: pname,
					RoleName:   aws.String(d.Id()),
				})
				if err != nil {
					if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
						log.Printf("[WARN] Inline role policy (%s) was already removed from role (%s)", aws.StringValue(pname), d.Id())
					} else {
						return fmt.Errorf("Error deleting inline policy of IAM Role %s: %s", d.Id(), err)
					}
				}
			}
		}
	}

	request := &iam.DeleteRoleInput{
		RoleName: aws.String(d.Id()),
	}

	// IAM is eventually consistent and deletion of attached policies may take time
	return resource.Retry(30*time.Second, func() *resource.RetryError {
		_, err := iamconn.DeleteRole(request)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "DeleteConflict" {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(fmt.Errorf("Error deleting IAM Role %s: %s", d.Id(), err))
		}
		return nil
	})
}
