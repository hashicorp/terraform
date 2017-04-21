package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSOpsworksPermission(t *testing.T) {
	sName := fmt.Sprintf("tf-ops-perm-%d", acctest.RandInt())
	var opsperm opsworks.Permission
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksPermissionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksPermissionCreate(sName, "true", "true", "iam_only"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksPermissionExists(
						"aws_opsworks_permission.tf-acc-perm", &opsperm),
					testAccCheckAWSOpsworksCreatePermissionAttributes(&opsperm, true, true, "iam_only"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_ssh", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_sudo", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "level", "iam_only",
					),
				),
			},
			{
				Config: testAccAwsOpsworksPermissionCreate(sName, "true", "false", "iam_only"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksPermissionExists(
						"aws_opsworks_permission.tf-acc-perm", &opsperm),
					testAccCheckAWSOpsworksCreatePermissionAttributes(&opsperm, true, false, "iam_only"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_ssh", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_sudo", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "level", "iam_only",
					),
				),
			},
			{
				Config: testAccAwsOpsworksPermissionCreate(sName, "false", "false", "deny"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksPermissionExists(
						"aws_opsworks_permission.tf-acc-perm", &opsperm),
					testAccCheckAWSOpsworksCreatePermissionAttributes(&opsperm, false, false, "deny"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_ssh", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_sudo", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "level", "deny",
					),
				),
			},
			{
				Config: testAccAwsOpsworksPermissionCreate(sName, "false", "false", "show"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksPermissionExists(
						"aws_opsworks_permission.tf-acc-perm", &opsperm),
					testAccCheckAWSOpsworksCreatePermissionAttributes(&opsperm, false, false, "show"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_ssh", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_sudo", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "level", "show",
					),
				),
			},
		},
	})
}

func testAccCheckAWSOpsworksPermissionExists(
	n string, opsperm *opsworks.Permission) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).opsworksconn

		params := &opsworks.DescribePermissionsInput{
			StackId:    aws.String(rs.Primary.Attributes["stack_id"]),
			IamUserArn: aws.String(rs.Primary.Attributes["user_arn"]),
		}
		resp, err := conn.DescribePermissions(params)

		if err != nil {
			return err
		}

		if v := len(resp.Permissions); v != 1 {
			return fmt.Errorf("Expected 1 response returned, got %d", v)
		}

		*opsperm = *resp.Permissions[0]

		return nil
	}
}

func testAccCheckAWSOpsworksCreatePermissionAttributes(
	opsperm *opsworks.Permission, allowSsh bool, allowSudo bool, level string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *opsperm.AllowSsh != allowSsh {
			return fmt.Errorf("Unnexpected allowSsh: %t", *opsperm.AllowSsh)
		}

		if *opsperm.AllowSudo != allowSudo {
			return fmt.Errorf("Unnexpected allowSudo: %t", *opsperm.AllowSudo)
		}

		if *opsperm.Level != level {
			return fmt.Errorf("Unnexpected level: %s", *opsperm.Level)
		}

		return nil
	}
}

func testAccCheckAwsOpsworksPermissionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AWSClient).opsworksconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_opsworks_permission" {
			continue
		}

		req := &opsworks.DescribePermissionsInput{
			IamUserArn: aws.String(rs.Primary.Attributes["user_arn"]),
		}

		resp, err := client.DescribePermissions(req)
		if err == nil {
			if len(resp.Permissions) > 0 {
				return fmt.Errorf("OpsWorks Permissions still exist.")
			}
		}

		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() != "ResourceNotFoundException" {
				return err
			}
		}
	}
	return nil
}

func testAccAwsOpsworksPermissionCreate(name, ssh, sudo, level string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_permission" "tf-acc-perm" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  allow_ssh = %s
  allow_sudo = %s
  user_arn = "${aws_opsworks_user_profile.user.user_arn}"
  level = "%s"
}

resource "aws_opsworks_user_profile" "user" {
  user_arn = "${aws_iam_user.user.arn}"
  ssh_username = "${aws_iam_user.user.name}"
}

resource "aws_iam_user" "user" {
	name = "%s"
	path = "/"
}
	
%s
`, ssh, sudo, level, name, testAccAwsOpsworksStackConfigVpcCreate(name))
}
