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
			{
				Config: testAccCheckAwsElbServiceAccountConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_service_account.main", "id", "797873946194"),
					resource.TestCheckResourceAttr("data.aws_elb_service_account.main", "arn", "arn:aws:iam::797873946194:root"),
				),
			},
			{
				Config: testAccCheckAwsElbServiceAccountExplicitRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_service_account.regional", "id", "156460612806"),
					resource.TestCheckResourceAttr("data.aws_elb_service_account.regional", "arn", "arn:aws:iam::156460612806:root"),
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
