package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamUserGroupMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserGroupMembershipCreate,
		Read:   resourceAwsIamUserGroupMembershipRead,
		Update: resourceAwsIamUserGroupMembershipUpdate,
		Delete: resourceAwsIamUserGroupMembershipDelete,

		Schema: map[string]*schema.Schema{
			"user": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"groups": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsIamUserGroupMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	user := d.Get("user").(string)
	groupList := expandStringList(d.Get("groups").(*schema.Set).List())

	if err := addUserToGroups(conn, user, groupList); err != nil {
		return err
	}

	d.SetId(resource.UniqueId())

	return resourceAwsIamUserGroupMembershipRead(d, meta)
}

func resourceAwsIamUserGroupMembershipRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	user := d.Get("user").(string)
	groups := d.Get("groups").(*schema.Set)
	var gl []string
	var marker *string

	for {
		resp, err := conn.ListGroupsForUser(&iam.ListGroupsForUserInput{
			UserName: &user,
			Marker:   marker,
		})
		if err != nil {
			if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
				// no such user
				log.Printf("[WARN] Groups not found for user (%s), removing from state", user)
				d.SetId("")
				return nil
			}
			return err
		}

		for _, g := range resp.Groups {
			// only read in the groups we care about
			if groups.Contains(*g.GroupName) {
				gl = append(gl, *g.GroupName)
			}
		}

		if !*resp.IsTruncated {
			break
		}

		marker = resp.Marker
	}

	if err := d.Set("groups", gl); err != nil {
		return fmt.Errorf("Error setting group list from IAM (%s), error: %s", user, err)
	}

	return nil
}

func resourceAwsIamUserGroupMembershipUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	if d.HasChange("groups") {
		user := d.Get("user").(string)

		o, n := d.GetChange("groups")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandStringList(os.Difference(ns).List())
		add := expandStringList(ns.Difference(os).List())

		if err := removeUserFromGroups(conn, user, remove); err != nil {
			return err
		}

		if err := addUserToGroups(conn, user, add); err != nil {
			return err
		}
	}

	return resourceAwsIamUserGroupMembershipRead(d, meta)
}

func resourceAwsIamUserGroupMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	user := d.Get("user").(string)
	groups := expandStringList(d.Get("groups").(*schema.Set).List())

	if err := removeUserFromGroups(conn, user, groups); err != nil {
		return err
	}

	return nil
}

func removeUserFromGroups(conn *iam.IAM, user string, groups []*string) error {
	for _, group := range groups {
		_, err := conn.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
			UserName:  &user,
			GroupName: group,
		})
		if err != nil {
			if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
				continue
			}
			return err
		}
	}

	return nil
}

func addUserToGroups(conn *iam.IAM, user string, groups []*string) error {
	for _, group := range groups {
		_, err := conn.AddUserToGroup(&iam.AddUserToGroupInput{
			UserName:  &user,
			GroupName: group,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
