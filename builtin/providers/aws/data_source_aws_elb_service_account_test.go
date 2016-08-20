package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSElbServiceAccount_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsElbServiceAccountConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_service_account.main", "id", "797873946194"),
				),
			},
			resource.TestStep{
				Config: testAccCheckAwsElbServiceAccountExplicitRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_service_account.regional", "id", "156460612806"),
				),
			},
		},
	})
}

const testAccCheckAwsElbServiceAccountConfig = `
data "aws_elb_service_account" "main" { }
`

const testAccCheckAwsElbServiceAccountExplicitRegionConfig = `
data "aws_elb_service_account" "regional" {
	region = "eu-west-1"
}
`
