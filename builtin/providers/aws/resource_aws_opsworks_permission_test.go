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
						"aws_opsworks_permission.tf-acc-perm", "allow_ssh", "1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "allow_sudo", "1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_permission.tf-acc-perm", "level", "iam_only",
					),
				),
			},
		},
	})
}

var testAccAwsOpsworksPermissionCreate = testAccAWSUserConfig + testAccAwsOpsworksStackConfigNoVpcCreate("tf-ops-acc-permission") + `
resource "aws_opsworks_permission" "tf-acc-perm" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  allow_ssh = true
  allow_sudo = true
  user_arn = "${aws_iam_user.user.arn}"
  level = "iam_only"
}
`
