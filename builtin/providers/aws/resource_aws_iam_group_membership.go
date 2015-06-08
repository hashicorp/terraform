package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamGroupMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamGroupMembershipCreate,
		Read:   resourceAwsIamGroupMembershipRead,
		//Update: resourceAwsIamGroupMembershipUpdate,
		Delete: resourceAwsIamGroupMembershipDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"users": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"group": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamGroupMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	userList := expandStringList(d.Get("users").(*schema.Set).List())
	group := d.Get("group").(string)

	for _, u := range userList {
		_, err := conn.AddUserToGroup(&iam.AddUserToGroupInput{
			UserName:  u,
			GroupName: aws.String(group),
		})

		if err != nil {
			return err
		}
	}

	d.SetId(d.Get("name").(string))
	return resourceAwsIamGroupMembershipRead(d, meta)
}

func resourceAwsIamGroupMembershipRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	resp, err := conn.GetGroup(&iam.GetGroupInput{
		GroupName: aws.String(d.Get("group").(string)),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// aws specific error
			log.Printf("\n\n------\n AWS Error: %s :::: %s", awsErr.Code(), awsErr.Message())
			// group not found
			d.SetId("")
		}
		return err
	}

	ul := make([]string, 0, len(resp.Users))
	for _, u := range resp.Users {
		ul = append(ul, *u.UserName)
	}

	if err := d.Set("users", ul); err != nil {
		return fmt.Errorf("[WARN] Error setting user list from IAM Group Membership (%s), error: %s", err)
	}

	return nil
}

func resourceAwsIamGroupMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	userList := expandStringList(d.Get("users").(*schema.Set).List())
	group := d.Get("group").(string)

	for _, u := range userList {
		_, err := conn.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
			UserName:  u,
			GroupName: aws.String(group),
		})

		if err != nil {
			return err
		}
	}

	d.SetId("")
	return nil
}
