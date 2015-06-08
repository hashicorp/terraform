package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamGroupMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamGroupMembershipCreate,
		Read:   resourceAwsIamGroupMembershipRead,
		//Update: resourceAwsIamGroupMembershipUpdate,
		Delete: resourceAwsIamGroupMembershipDelete,

		Schema: map[string]*schema.Schema{
			"user_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamGroupMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	_, err := conn.AddUserToGroup(&iam.AddUserToGroupInput{
		UserName:  aws.String(d.Get("user_name").(string)),
		GroupName: aws.String(d.Get("group_name").(string)),
	})

	if err != nil {
		return err
	}

	d.SetId(resource.UniqueId())
	return resourceAwsIamGroupMembershipRead(d, meta)
}

func resourceAwsIamGroupMembershipRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	u := d.Get("user_name").(string)
	resp, err := conn.ListGroupsForUser(&iam.ListGroupsForUserInput{
		UserName: aws.String(u),
	})

	if err != nil {
		return err
	}

	d.Set("user_name", u)

	gn := d.Get("group_name").(string)
	var group *iam.Group
	for _, g := range resp.Groups {
		if gn == *g.GroupName {
			group = g
		}
	}

	if group == nil {
		// if not found, set to ""
		log.Printf("[DEBUG] Group (%s) not found for User (%s)", u, gn)
		d.SetId("")
	}

	return nil
}

func resourceAwsIamGroupMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	_, err := conn.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
		UserName:  aws.String(d.Get("user_name").(string)),
		GroupName: aws.String(d.Get("group_name").(string)),
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
