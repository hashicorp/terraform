package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSGroupMembership_basic(t *testing.T) {
	var group iam.Group

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupMembershipDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSGroupMemberConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupMembershipExists("aws_iam_group_membership.team", &group),
					testAccCheckAWSGroupMembershipAttributes(&group),
				),
			},
		},
	})
}

func testAccCheckAWSGroupMembershipDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_group_membership" {
			continue
		}

		// Try to get user
		user := rs.Primary.Attributes["user_name"]
		group := rs.Primary.Attributes["group_name"]

		resp, err := iamconn.ListGroupsForUser(&iam.ListGroupsForUserInput{
			UserName: aws.String(user),
		})
		if err != nil {
			// might error here
			return err
		}

		for _, g := range resp.Groups {
			if group == *g.GroupName {
				return fmt.Errorf("Error: User (%s) is still a memeber of Group (%s)", user, group)
			}
		}

	}

	return nil
}

func testAccCheckAWSGroupMembershipExists(n string, g *iam.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No User name is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		user := rs.Primary.Attributes["user_name"]
		gn := rs.Primary.Attributes["group_name"]

		resp, err := iamconn.ListGroupsForUser(&iam.ListGroupsForUserInput{
			UserName: aws.String(user),
		})
		if err != nil {
			return err
		}

		for _, i := range resp.Groups {
			if gn == *i.GroupName {
				*g = *i
				return nil
			}
		}

		return fmt.Errorf("Error: User (%s) not a member of Group (%s)", user, gn)
	}
}

func testAccCheckAWSGroupMembershipAttributes(group *iam.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.GroupName != "test-group" {
			return fmt.Errorf("Bad group membership: expected %s, got %s", "test-group-update", *group.GroupName)
		}
		return nil
	}
}

const testAccAWSGroupMemberConfig = `
resource "aws_iam_group" "group" {
	name = "test-group"
	path = "/"
}

resource "aws_iam_user" "user" {
	name = "test-user"
	path = "/"
}

resource "aws_iam_group_membership" "team" {
	user_name = "${aws_iam_user.user.name}"
	group_name = "${aws_iam_group.group.name}"
}
`
