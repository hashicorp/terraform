package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsIamUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserCreate,
		Read:   resourceAwsIamUserRead,
		Update: resourceAwsIamUserUpdate,
		Delete: resourceAwsIamUserDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			/*
				The UniqueID could be used as the Id(), but none of the API
				calls allow specifying a user by the UniqueID: they require the
				name. The only way to locate a user by UniqueID is to list them
				all and that would make this provider unnecessarily complex
				and inefficient. Still, there are other reasons one might want
				the UniqueID, so we can make it available.
			*/
			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAwsIamUserName,
			},
			"path": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
			},
			"permissions_boundary": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 2048),
			},
			"force_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Delete user even if it has non-Terraform-managed IAM access keys, login profile or MFA devices",
			},
		},
	}
}

func resourceAwsIamUserCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	path := d.Get("path").(string)

	request := &iam.CreateUserInput{
		Path:     aws.String(path),
		UserName: aws.String(name),
	}

	if v, ok := d.GetOk("permissions_boundary"); ok && v.(string) != "" {
		request.PermissionsBoundary = aws.String(v.(string))
	}

	log.Println("[DEBUG] Create IAM User request:", request)
	createResp, err := iamconn.CreateUser(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM User %s: %s", name, err)
	}

	d.SetId(aws.StringValue(createResp.User.UserName))

	return resourceAwsIamUserRead(d, meta)
}

func resourceAwsIamUserRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.GetUserInput{
		UserName: aws.String(d.Id()),
	}

	output, err := iamconn.GetUser(request)
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			log.Printf("[WARN] No IAM user by name (%s) found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM User %s: %s", d.Id(), err)
	}

	if output == nil || output.User == nil {
		log.Printf("[WARN] No IAM user by name (%s) found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", output.User.Arn)
	d.Set("name", output.User.UserName)
	d.Set("path", output.User.Path)
	if output.User.PermissionsBoundary != nil {
		d.Set("permissions_boundary", output.User.PermissionsBoundary.PermissionsBoundaryArn)
	}
	d.Set("unique_id", output.User.UserId)

	return nil
}

func resourceAwsIamUserUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if d.HasChange("name") || d.HasChange("path") {
		on, nn := d.GetChange("name")
		_, np := d.GetChange("path")

		request := &iam.UpdateUserInput{
			UserName:    aws.String(on.(string)),
			NewUserName: aws.String(nn.(string)),
			NewPath:     aws.String(np.(string)),
		}

		log.Println("[DEBUG] Update IAM User request:", request)
		_, err := iamconn.UpdateUser(request)
		if err != nil {
			if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
				log.Printf("[WARN] No IAM user by name (%s) found", d.Id())
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error updating IAM User %s: %s", d.Id(), err)
		}

		d.SetId(nn.(string))
	}

	if d.HasChange("permissions_boundary") {
		permissionsBoundary := d.Get("permissions_boundary").(string)
		if permissionsBoundary != "" {
			input := &iam.PutUserPermissionsBoundaryInput{
				PermissionsBoundary: aws.String(permissionsBoundary),
				UserName:            aws.String(d.Id()),
			}
			_, err := iamconn.PutUserPermissionsBoundary(input)
			if err != nil {
				return fmt.Errorf("error updating IAM User permissions boundary: %s", err)
			}
		} else {
			input := &iam.DeleteUserPermissionsBoundaryInput{
				UserName: aws.String(d.Id()),
			}
			_, err := iamconn.DeleteUserPermissionsBoundary(input)
			if err != nil {
				return fmt.Errorf("error deleting IAM User permissions boundary: %s", err)
			}
		}
	}

	return resourceAwsIamUserRead(d, meta)
}

func resourceAwsIamUserDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	// IAM Users must be removed from all groups before they can be deleted
	var groups []string
	listGroups := &iam.ListGroupsForUserInput{
		UserName: aws.String(d.Id()),
	}
	pageOfGroups := func(page *iam.ListGroupsForUserOutput, lastPage bool) (shouldContinue bool) {
		for _, g := range page.Groups {
			groups = append(groups, *g.GroupName)
		}
		return !lastPage
	}
	err := iamconn.ListGroupsForUserPages(listGroups, pageOfGroups)
	if err != nil {
		return fmt.Errorf("Error removing user %q from all groups: %s", d.Id(), err)
	}
	for _, g := range groups {
		// use iam group membership func to remove user from all groups
		log.Printf("[DEBUG] Removing IAM User %s from IAM Group %s", d.Id(), g)
		if err := removeUsersFromGroup(iamconn, []*string{aws.String(d.Id())}, g); err != nil {
			return err
		}
	}

	// All access keys, MFA devices and login profile for the user must be removed
	if d.Get("force_destroy").(bool) {
		var accessKeys []string
		listAccessKeys := &iam.ListAccessKeysInput{
			UserName: aws.String(d.Id()),
		}
		pageOfAccessKeys := func(page *iam.ListAccessKeysOutput, lastPage bool) (shouldContinue bool) {
			for _, k := range page.AccessKeyMetadata {
				accessKeys = append(accessKeys, *k.AccessKeyId)
			}
			return !lastPage
		}
		err = iamconn.ListAccessKeysPages(listAccessKeys, pageOfAccessKeys)
		if err != nil {
			return fmt.Errorf("Error removing access keys of user %s: %s", d.Id(), err)
		}
		for _, k := range accessKeys {
			_, err := iamconn.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				UserName:    aws.String(d.Id()),
				AccessKeyId: aws.String(k),
			})
			if err != nil {
				return fmt.Errorf("Error deleting access key %s: %s", k, err)
			}
		}

		var MFADevices []string
		listMFADevices := &iam.ListMFADevicesInput{
			UserName: aws.String(d.Id()),
		}
		pageOfMFADevices := func(page *iam.ListMFADevicesOutput, lastPage bool) (shouldContinue bool) {
			for _, m := range page.MFADevices {
				MFADevices = append(MFADevices, *m.SerialNumber)
			}
			return !lastPage
		}
		err = iamconn.ListMFADevicesPages(listMFADevices, pageOfMFADevices)
		if err != nil {
			return fmt.Errorf("Error removing MFA devices of user %s: %s", d.Id(), err)
		}
		for _, m := range MFADevices {
			_, err := iamconn.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
				UserName:     aws.String(d.Id()),
				SerialNumber: aws.String(m),
			})
			if err != nil {
				return fmt.Errorf("Error deactivating MFA device %s: %s", m, err)
			}
		}

		err = resource.Retry(1*time.Minute, func() *resource.RetryError {
			_, err = iamconn.DeleteLoginProfile(&iam.DeleteLoginProfileInput{
				UserName: aws.String(d.Id()),
			})
			if err != nil {
				if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
					return nil
				}
				// EntityTemporarilyUnmodifiable: Login Profile for User XXX cannot be modified while login profile is being created.
				if isAWSErr(err, iam.ErrCodeEntityTemporarilyUnmodifiableException, "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("Error deleting Account Login Profile: %s", err)
		}
	}

	deleteUserInput := &iam.DeleteUserInput{
		UserName: aws.String(d.Id()),
	}

	log.Println("[DEBUG] Delete IAM User request:", deleteUserInput)
	_, err = iamconn.DeleteUser(deleteUserInput)
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return nil
		}
		return fmt.Errorf("Error deleting IAM User %s: %s", d.Id(), err)
	}

	return nil
}

func validateAwsIamUserName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z=,.@\-_+]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters, hyphens, underscores, commas, periods, @ symbols, plus and equals signs allowed in %q: %q",
			k, value))
	}
	return
}
