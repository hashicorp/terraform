package aws

import (
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamInstanceProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamInstanceProfileCreate,
		Read:   resourceAwsIamInstanceProfileRead,
		Update: resourceAwsIamInstanceProfileUpdate,
		Delete: resourceAwsIamInstanceProfileDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"create_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/2485f5c/botocore/data/iam/2010-05-08/service-2.json#L8196-L8201
					value := v.(string)
					if len(value) > 128 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 128 characters", k))
					}
					if !regexp.MustCompile("^[\\w+=,.@-]+$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must match [\\w+=,.@-]", k))
					}
					return
				},
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},
			"roles": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsIamInstanceProfileCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

	request := &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(name),
		Path:                aws.String(d.Get("path").(string)),
	}

	var err error
	response, err := iamconn.CreateInstanceProfile(request)
	if err == nil {
		err = instanceProfileReadResult(d, response.InstanceProfile)
	}
	if err != nil {
		return fmt.Errorf("Error creating IAM instance profile %s: %s", name, err)
	}

	return instanceProfileSetRoles(d, iamconn)
}

func instanceProfileAddRole(iamconn *iam.IAM, profileName, roleName string) error {
	request := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		RoleName:            aws.String(roleName),
	}

	_, err := iamconn.AddRoleToInstanceProfile(request)
	return err
}

func instanceProfileRemoveRole(iamconn *iam.IAM, profileName, roleName string) error {
	request := &iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		RoleName:            aws.String(roleName),
	}

	_, err := iamconn.RemoveRoleFromInstanceProfile(request)
	if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
		return nil
	}
	return err
}

func instanceProfileSetRoles(d *schema.ResourceData, iamconn *iam.IAM) error {
	oldInterface, newInterface := d.GetChange("roles")
	oldRoles := oldInterface.(*schema.Set)
	newRoles := newInterface.(*schema.Set)

	currentRoles := schema.CopySet(oldRoles)

	d.Partial(true)

	for _, role := range oldRoles.Difference(newRoles).List() {
		err := instanceProfileRemoveRole(iamconn, d.Id(), role.(string))
		if err != nil {
			return fmt.Errorf("Error removing role %s from IAM instance profile %s: %s", role, d.Id(), err)
		}
		currentRoles.Remove(role)
		d.Set("roles", currentRoles)
		d.SetPartial("roles")
	}

	for _, role := range newRoles.Difference(oldRoles).List() {
		err := instanceProfileAddRole(iamconn, d.Id(), role.(string))
		if err != nil {
			return fmt.Errorf("Error adding role %s to IAM instance profile %s: %s", role, d.Id(), err)
		}
		currentRoles.Add(role)
		d.Set("roles", currentRoles)
		d.SetPartial("roles")
	}

	d.Partial(false)

	return nil
}

func instanceProfileRemoveAllRoles(d *schema.ResourceData, iamconn *iam.IAM) error {
	for _, role := range d.Get("roles").(*schema.Set).List() {
		err := instanceProfileRemoveRole(iamconn, d.Id(), role.(string))
		if err != nil {
			return fmt.Errorf("Error removing role %s from IAM instance profile %s: %s", role, d.Id(), err)
		}
	}
	return nil
}

func resourceAwsIamInstanceProfileUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if !d.HasChange("roles") {
		return nil
	}

	return instanceProfileSetRoles(d, iamconn)
}

func resourceAwsIamInstanceProfileRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(d.Id()),
	}

	result, err := iamconn.GetInstanceProfile(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM instance profile %s: %s", d.Id(), err)
	}

	return instanceProfileReadResult(d, result.InstanceProfile)
}

func resourceAwsIamInstanceProfileDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if err := instanceProfileRemoveAllRoles(d, iamconn); err != nil {
		return err
	}

	request := &iam.DeleteInstanceProfileInput{
		InstanceProfileName: aws.String(d.Id()),
	}
	_, err := iamconn.DeleteInstanceProfile(request)
	if err != nil {
		return fmt.Errorf("Error deleting IAM instance profile %s: %s", d.Id(), err)
	}
	d.SetId("")
	return nil
}

func instanceProfileReadResult(d *schema.ResourceData, result *iam.InstanceProfile) error {
	d.SetId(*result.InstanceProfileName)
	if err := d.Set("name", result.InstanceProfileName); err != nil {
		return err
	}
	if err := d.Set("arn", result.Arn); err != nil {
		return err
	}
	if err := d.Set("path", result.Path); err != nil {
		return err
	}

	roles := &schema.Set{F: schema.HashString}
	for _, role := range result.Roles {
		roles.Add(*role.RoleName)
	}
	if err := d.Set("roles", roles); err != nil {
		return err
	}

	return nil
}
