package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSOpsworksUserProfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksUserProfileCreate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_user_profile.user", "ssh_public_key", "",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_user_profile.user", "ssh_username", "test-user",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_user_profile.user", "allow_self_management", "false",
					),
				),
			},
		},
	})
}

var testAccAwsOpsworksUserProfileCreate = testAccAWSOpsUserConfig + testAccAwsOpsworksStackConfigNoVpcCreate("tf-ops-acc-user-profile") + `
resource "aws_opsworks_user_profile" "user" {
  user_arn = "${aws_iam_user.user.arn}"
  ssh_username = "${aws_iam_user.user.name}"
}
`

var testAccAWSOpsUserConfig = `
resource "aws_iam_user" "user" {
	name = "test-user"
	path = "/"
}
`
