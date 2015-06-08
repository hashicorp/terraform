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
	conn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_group_membership" {
			continue
		}

		// Try to get user
		group := rs.Primary.Attributes["group"]

		_, err := conn.GetGroup(&iam.GetGroupInput{
			GroupName: aws.String(group),
		})
		if err != nil {
			// might error here
			return err
		}

		return fmt.Errorf("Error: Group (%s) still exists", group)

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

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		gn := rs.Primary.Attributes["group"]

		resp, err := conn.GetGroup(&iam.GetGroupInput{
			GroupName: aws.String(gn),
		})

		if err != nil {
			return fmt.Errorf("Error: Group (%s) not found", gn)
		}

		*g = *resp.Group

		return nil
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
	name = "tf-testing-group-membership"
	users = ["${aws_iam_user.user.name}"]
	group = "${aws_iam_group.group.name}"
}
`
