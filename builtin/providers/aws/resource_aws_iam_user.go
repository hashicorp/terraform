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
		// There is an UpdateUser API call, but goamz doesn't support it yet.
		// XXX but we aren't using goamz anymore.
		//Update: resourceAwsIamUserUpdate,
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
				ForceNew: true,
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

	request := &iam.CreateUserInput{
		Path:     aws.String(d.Get("path").(string)),
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

	request := &iam.GetUserInput{
		UserName: aws.String(d.Id()),
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
	if err := d.Set("arn", user.ARN); err != nil {
		return err
	}
	if err := d.Set("path", user.Path); err != nil {
		return err
	}
	if err := d.Set("unique_id", user.UserID); err != nil {
		return err
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
