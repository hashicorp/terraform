package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
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
			"arn": &schema.Schema{
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
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAwsIamUserName,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},
			"force_destroy": &schema.Schema{
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

	log.Println("[DEBUG] Create IAM User request:", request)
	createResp, err := iamconn.CreateUser(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM User %s: %s", name, err)
	}
	d.SetId(*createResp.User.UserName)
	return resourceAwsIamUserReadResult(d, createResp.User)
}

func resourceAwsIamUserRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.GetUserInput{
		UserName: aws.String(d.Id()),
	}

	getResp, err := iamconn.GetUser(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX test me
			log.Printf("[WARN] No IAM user by name (%s) found", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM User %s: %s", d.Id(), err)
	}
	return resourceAwsIamUserReadResult(d, getResp.User)
}

func resourceAwsIamUserReadResult(d *schema.ResourceData, user *iam.User) error {
	if err := d.Set("name", user.UserName); err != nil {
		return err
	}
	if err := d.Set("arn", user.Arn); err != nil {
		return err
	}
	if err := d.Set("path", user.Path); err != nil {
		return err
	}
	if err := d.Set("unique_id", user.UserId); err != nil {
		return err
	}
	return nil
}

func resourceAwsIamUserUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("name") || d.HasChange("path") {
		iamconn := meta.(*AWSClient).iamconn
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
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] No IAM user by name (%s) found", d.Id())
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error updating IAM User %s: %s", d.Id(), err)
		}
		return resourceAwsIamUserRead(d, meta)
	}
	return nil
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

		_, err = iamconn.DeleteLoginProfile(&iam.DeleteLoginProfileInput{
			UserName: aws.String(d.Id()),
		})
		if err != nil {
			if iamerr, ok := err.(awserr.Error); !ok || iamerr.Code() != "NoSuchEntity" {
				return fmt.Errorf("Error deleting Account Login Profile: %s", err)
			}
		}
	}

	request := &iam.DeleteUserInput{
		UserName: aws.String(d.Id()),
	}

	log.Println("[DEBUG] Delete IAM User request:", request)
	if _, err := iamconn.DeleteUser(request); err != nil {
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
