package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSOpsworksPermission(t *testing.T) {
	sName := fmt.Sprintf("tf-ops-perm-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksPermissionCreate(sName, "true", "true", "iam_only"),
				Check: resource.ComposeTestCheckFunc(
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
			resource.TestStep{
				Config: testAccAwsOpsworksPermissionCreate(sName, "true", "false", "iam_only"),
				Check: resource.ComposeTestCheckFunc(
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
			resource.TestStep{
				Config: testAccAwsOpsworksPermissionCreate(sName, "false", "false", "deny"),
				Check: resource.ComposeTestCheckFunc(
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
			resource.TestStep{
				Config: testAccAwsOpsworksPermissionCreate(sName, "false", "false", "show"),
				Check: resource.ComposeTestCheckFunc(
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
