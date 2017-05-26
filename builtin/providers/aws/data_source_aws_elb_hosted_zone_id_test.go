package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSElbHostedZoneId_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsElbHostedZoneIdConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_hosted_zone_id.main", "id", "Z1H1FL5HABSF5"),
				),
			},
			{
				Config: testAccCheckAwsElbHostedZoneIdExplicitRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elb_hosted_zone_id.regional", "id", "Z32O12XQLNTSW2"),
				),
			},
		},
	})
}

const testAccCheckAwsElbHostedZoneIdConfig = `
data "aws_elb_hosted_zone_id" "main" { }
`

const testAccCheckAwsElbHostedZoneIdExplicitRegionConfig = `
data "aws_elb_hosted_zone_id" "regional" {
	region = "eu-west-1"
}
`
