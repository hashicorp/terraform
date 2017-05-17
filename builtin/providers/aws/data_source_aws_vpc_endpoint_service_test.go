package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsVpcEndpointService(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpcEndpointServiceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcEndpointServiceCheck("data.aws_vpc_endpoint_service.s3"),
				),
			},
		},
	})
}

func testAccDataSourceAwsVpcEndpointServiceCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		attr := rs.Primary.Attributes

		name := attr["service_name"]
		if name != "com.amazonaws.us-west-2.s3" {
			return fmt.Errorf("bad service name %s", name)
		}

		return nil
	}
}

const testAccDataSourceAwsVpcEndpointServiceConfig = `
provider "aws" {
  region = "us-west-2"
}

data "aws_vpc_endpoint_service" "s3" {
  service = "s3"
}
`
