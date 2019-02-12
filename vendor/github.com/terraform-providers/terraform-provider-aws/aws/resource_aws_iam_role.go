package aws

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
					if !regexp.MustCompile(`^[\w+=,.@-]*$`).MatchString(value) {
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
					if !regexp.MustCompile(`^[\w+=,.@-]*$`).MatchString(value) {
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

			"permissions_boundary": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 2048),
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
				ValidateFunc:     validation.ValidateJsonString,
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

			"tags": tagsSchema(),
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

	if v, ok := d.GetOk("permissions_boundary"); ok {
		request.PermissionsBoundary = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		request.Tags = tagsFromMapIAM(v.(map[string]interface{}))
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
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			log.Printf("[WARN] IAM Role %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM Role %s: %s", d.Id(), err)
	}

	if getResp == nil || getResp.Role == nil {
		log.Printf("[WARN] IAM Role %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	role := getResp.Role

	d.Set("arn", role.Arn)
	if err := d.Set("create_date", role.CreateDate.Format(time.RFC3339)); err != nil {
		return err
	}
	d.Set("description", role.Description)
	d.Set("max_session_duration", role.MaxSessionDuration)
	d.Set("name", role.RoleName)
	d.Set("path", role.Path)
	if role.PermissionsBoundary != nil {
		d.Set("permissions_boundary", role.PermissionsBoundary.PermissionsBoundaryArn)
	}
	d.Set("unique_id", role.RoleId)
	if err := d.Set("tags", tagsToMapIAM(role.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
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
			if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
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
			if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
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

	if d.HasChange("permissions_boundary") {
		permissionsBoundary := d.Get("permissions_boundary").(string)
		if permissionsBoundary != "" {
			input := &iam.PutRolePermissionsBoundaryInput{
				PermissionsBoundary: aws.String(permissionsBoundary),
				RoleName:            aws.String(d.Id()),
			}
			_, err := iamconn.PutRolePermissionsBoundary(input)
			if err != nil {
				return fmt.Errorf("error updating IAM Role permissions boundary: %s", err)
			}
		} else {
			input := &iam.DeleteRolePermissionsBoundaryInput{
				RoleName: aws.String(d.Id()),
			}
			_, err := iamconn.DeleteRolePermissionsBoundary(input)
			if err != nil {
				return fmt.Errorf("error deleting IAM Role permissions boundary: %s", err)
			}
		}
	}

	if d.HasChange("tags") {
		// Reset all tags to empty set
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		c, r := diffTagsIAM(tagsFromMapIAM(o), tagsFromMapIAM(n))

		if len(r) > 0 {
			_, err := iamconn.UntagRole(&iam.UntagRoleInput{
				RoleName: aws.String(d.Id()),
				TagKeys:  tagKeysIam(r),
			})
			if err != nil {
				return fmt.Errorf("error deleting IAM role tags: %s", err)
			}
		}

		if len(c) > 0 {
			input := &iam.TagRoleInput{
				RoleName: aws.String(d.Id()),
				Tags:     c,
			}
			_, err := iamconn.TagRole(input)
			if err != nil {
				return fmt.Errorf("error update IAM role tags: %s", err)
			}
		}
	}

	return resourceAwsIamRoleRead(d, meta)
}

func resourceAwsIamRoleDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	// Roles cannot be destroyed when attached to an existing Instance Profile
	if err := deleteAwsIamRoleInstanceProfiles(iamconn, d.Id()); err != nil {
		return fmt.Errorf("error deleting IAM Role (%s) instance profiles: %s", d.Id(), err)
	}

	if d.Get("force_detach_policies").(bool) {
		// For managed policies
		if err := deleteAwsIamRolePolicyAttachments(iamconn, d.Id()); err != nil {
			return fmt.Errorf("error deleting IAM Role (%s) policy attachments: %s", d.Id(), err)
		}

		// For inline policies
		if err := deleteAwsIamRolePolicies(iamconn, d.Id()); err != nil {
			return fmt.Errorf("error deleting IAM Role (%s) policies: %s", d.Id(), err)
		}
	}

	deleteRoleInput := &iam.DeleteRoleInput{
		RoleName: aws.String(d.Id()),
	}

	// IAM is eventually consistent and deletion of attached policies may take time
	return resource.Retry(30*time.Second, func() *resource.RetryError {
		_, err := iamconn.DeleteRole(deleteRoleInput)
		if err != nil {
			if isAWSErr(err, iam.ErrCodeDeleteConflictException, "") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(fmt.Errorf("Error deleting IAM Role %s: %s", d.Id(), err))
		}
		return nil
	})
}

func deleteAwsIamRoleInstanceProfiles(conn *iam.IAM, rolename string) error {
	resp, err := conn.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(rolename),
	})
	if err != nil {
		return fmt.Errorf("Error listing Profiles for IAM Role (%s) when trying to delete: %s", rolename, err)
	}

	// Loop and remove this Role from any Profiles
	for _, i := range resp.InstanceProfiles {
		input := &iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: i.InstanceProfileName,
			RoleName:            aws.String(rolename),
		}

		_, err := conn.RemoveRoleFromInstanceProfile(input)

		if err != nil {
			return fmt.Errorf("Error deleting IAM Role %s: %s", rolename, err)
		}
	}

	return nil
}

func deleteAwsIamRolePolicyAttachments(conn *iam.IAM, rolename string) error {
	managedPolicies := make([]*string, 0)
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(rolename),
	}

	err := conn.ListAttachedRolePoliciesPages(input, func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
		for _, v := range page.AttachedPolicies {
			managedPolicies = append(managedPolicies, v.PolicyArn)
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("Error listing Policies for IAM Role (%s) when trying to delete: %s", rolename, err)
	}
	for _, parn := range managedPolicies {
		input := &iam.DetachRolePolicyInput{
			PolicyArn: parn,
			RoleName:  aws.String(rolename),
		}

		_, err = conn.DetachRolePolicy(input)

		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			continue
		}

		if err != nil {
			return fmt.Errorf("Error deleting IAM Role %s: %s", rolename, err)
		}
	}

	return nil
}

func deleteAwsIamRolePolicies(conn *iam.IAM, rolename string) error {
	inlinePolicies := make([]*string, 0)
	input := &iam.ListRolePoliciesInput{
		RoleName: aws.String(rolename),
	}

	err := conn.ListRolePoliciesPages(input, func(page *iam.ListRolePoliciesOutput, lastPage bool) bool {
		inlinePolicies = append(inlinePolicies, page.PolicyNames...)
		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("Error listing inline Policies for IAM Role (%s) when trying to delete: %s", rolename, err)
	}

	for _, pname := range inlinePolicies {
		input := &iam.DeleteRolePolicyInput{
			PolicyName: pname,
			RoleName:   aws.String(rolename),
		}

		_, err := conn.DeleteRolePolicy(input)

		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			continue
		}

		if err != nil {
			return fmt.Errorf("Error deleting inline policy of IAM Role %s: %s", rolename, err)
		}
	}

	return nil
}
