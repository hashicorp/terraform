package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSOpsworksUserProfile(t *testing.T) {
	rName := fmt.Sprintf("test-user-%d", acctest.RandInt())
	roleName := fmt.Sprintf("tf-ops-user-profile-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksUserProfileCreate(rName, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_user_profile.user", "ssh_public_key", "",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_user_profile.user", "ssh_username", rName,
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_user_profile.user", "allow_self_management", "false",
					),
				),
			},
		},
	})
}

func testAccAwsOpsworksUserProfileCreate(rn, roleName string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_user_profile" "user" {
  user_arn = "${aws_iam_user.user.arn}"
  ssh_username = "${aws_iam_user.user.name}"
}

resource "aws_iam_user" "user" {
	name = "%s"
	path = "/"
}

%s
	`, rn, testAccAwsOpsworksStackConfigNoVpcCreate(roleName))
}
