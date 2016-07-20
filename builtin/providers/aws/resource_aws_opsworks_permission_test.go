package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSOpsworksPermission(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksPermissionCreate,
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
		},
	})
}

var testAccAwsOpsworksPermissionCreate = testAccAwsOpsworksUserProfileCreate + `
resource "aws_opsworks_permission" "tf-acc-perm" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  allow_ssh = true
  allow_sudo = true
  user_arn = "${aws_opsworks_user_profile.user.user_arn}"
  level = "iam_only"
}
`
