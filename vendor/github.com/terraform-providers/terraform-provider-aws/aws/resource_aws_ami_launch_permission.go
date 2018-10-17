package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAmiLaunchPermission() *schema.Resource {
	return &schema.Resource{
		Exists: resourceAwsAmiLaunchPermissionExists,
		Create: resourceAwsAmiLaunchPermissionCreate,
		Read:   resourceAwsAmiLaunchPermissionRead,
		Delete: resourceAwsAmiLaunchPermissionDelete,

		Schema: map[string]*schema.Schema{
			"image_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"account_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsAmiLaunchPermissionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*AWSClient).ec2conn

	image_id := d.Get("image_id").(string)
	account_id := d.Get("account_id").(string)
	return hasLaunchPermission(conn, image_id, account_id)
}

func resourceAwsAmiLaunchPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	image_id := d.Get("image_id").(string)
	account_id := d.Get("account_id").(string)

	_, err := conn.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
		ImageId:   aws.String(image_id),
		Attribute: aws.String("launchPermission"),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: []*ec2.LaunchPermission{
				{UserId: aws.String(account_id)},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error creating ami launch permission: %s", err)
	}

	d.SetId(fmt.Sprintf("%s-%s", image_id, account_id))
	return nil
}

func resourceAwsAmiLaunchPermissionRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsAmiLaunchPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	image_id := d.Get("image_id").(string)
	account_id := d.Get("account_id").(string)

	_, err := conn.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
		ImageId:   aws.String(image_id),
		Attribute: aws.String("launchPermission"),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Remove: []*ec2.LaunchPermission{
				{UserId: aws.String(account_id)},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error removing ami launch permission: %s", err)
	}

	return nil
}

func hasLaunchPermission(conn *ec2.EC2, image_id string, account_id string) (bool, error) {
	attrs, err := conn.DescribeImageAttribute(&ec2.DescribeImageAttributeInput{
		ImageId:   aws.String(image_id),
		Attribute: aws.String("launchPermission"),
	})
	if err != nil {
		// When an AMI disappears out from under a launch permission resource, we will
		// see either InvalidAMIID.NotFound or InvalidAMIID.Unavailable.
		if ec2err, ok := err.(awserr.Error); ok && strings.HasPrefix(ec2err.Code(), "InvalidAMIID") {
			log.Printf("[DEBUG] %s no longer exists, so we'll drop launch permission for %s from the state", image_id, account_id)
			return false, nil
		}
		return false, err
	}

	for _, lp := range attrs.LaunchPermissions {
		if *lp.UserId == account_id {
			return true, nil
		}
	}
	return false, nil
}
