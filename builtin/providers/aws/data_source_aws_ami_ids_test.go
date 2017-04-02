package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAwsAmiIds_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsAmiIdsConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDataSourceID("data.aws_ami_ids.ubuntu"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsAmiIds_empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsAmiIdsConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDataSourceID("data.aws_ami_ids.empty"),
					resource.TestCheckResourceAttr("data.aws_ami_ids.empty", "ids.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceAwsAmiIdsConfig_basic = `
data "aws_ami_ids" "ubuntu" {
  owners = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/images/ubuntu-*-*-amd64-server-*"]
  }
}
`

const testAccDataSourceAwsAmiIdsConfig_empty = `
data "aws_ami_ids" "empty" {
  filter {
    name   = "name"
    values = []
  }
}
`
