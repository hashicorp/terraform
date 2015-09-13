package aws

import (
	"fmt"

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

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			/*
				The UniqueID could be used as the Id(), but none of the API
				calls allow specifying a user by the UniqueID: they require the
				name. The only way to locate a user by UniqueID is to list them
				all and that would make this provider unnecessarilly complex
				and inefficient. Still, there are other reasons one might want
				the UniqueID, so we can make it availible.
			*/
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
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

	createResp, err := iamconn.CreateUser(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM User %s: %s", name, err)
	}
	return resourceAwsIamUserReadResult(d, createResp.User)
}

func resourceAwsIamUserRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	request := &iam.GetUserInput{
		UserName: aws.String(name),
	}

	getResp, err := iamconn.GetUser(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX test me
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM User %s: %s", d.Id(), err)
	}
	return resourceAwsIamUserReadResult(d, getResp.User)
}

func resourceAwsIamUserReadResult(d *schema.ResourceData, user *iam.User) error {
	d.SetId(*user.UserName)
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
		op, np := d.GetChange("path")
		fmt.Println(on, nn, op, np)
		request := &iam.UpdateUserInput{
			UserName:    aws.String(on.(string)),
			NewUserName: aws.String(nn.(string)),
			NewPath:     aws.String(np.(string)),
		}
		_, err := iamconn.UpdateUser(request)
		if err != nil {
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
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

	request := &iam.DeleteUserInput{
		UserName: aws.String(d.Id()),
	}

	if _, err := iamconn.DeleteUser(request); err != nil {
		return fmt.Errorf("Error deleting IAM User %s: %s", d.Id(), err)
	}
	return nil
}
