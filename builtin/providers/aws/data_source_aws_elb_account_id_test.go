package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSElbAccountId_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsElbAccountIdConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_account_id.main", "id", "797873946194"),
				),
			},
			resource.TestStep{
				Config: testAccCheckAwsElbAccountIdExplicitRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_account_id.regional", "id", "156460612806"),
				),
			},
		},
	})
}

const testAccCheckAwsElbAccountIdConfig = `
data "aws_elb_account_id" "main" { }
`

const testAccCheckAwsElbAccountIdExplicitRegionConfig = `
data "aws_elb_account_id" "regional" {
	region = "eu-west-1"
}
`
