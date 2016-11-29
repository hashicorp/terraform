package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSOpsworksPermission(t *testing.T) {
	rName := fmt.Sprintf("test-user-%d", acctest.RandInt())
	roleName := fmt.Sprintf("tf-ops-user-profile-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksPermissionCreate(rName, roleName),
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

func testAccAwsOpsworksPermissionCreate(rn, roleName string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_permission" "tf-acc-perm" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  allow_ssh = true
  allow_sudo = true
  user_arn = "${aws_opsworks_user_profile.user.user_arn}"
  level = "iam_only"
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
`, rn, testAccAwsOpsworksStackConfigNoVpcCreate(rn))
}
