package aws

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsVpcEndpointServices(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpcEndpointServicesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcEndpointServicesCheck("data.aws_vpc_endpoint_services.endpoint_services"),
				),
			},
		},
	})
}

func testAccDataSourceAwsVpcEndpointServicesCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		attr := rs.Primary.Attributes

		var (
			n   int
			err error
		)

		if n, err = strconv.Atoi(attr["names.#"]); err != nil {
			return err
		}
		if n < 1 {
			return fmt.Errorf("Number of services seem suspiciously low: %d", n)
		}

		return nil
	}
}

const testAccDataSourceAwsVpcEndpointServicesConfig = `
provider "aws" {
  region = "us-west-2"
}

data "aws_vpc_endpoint_services" "endpoint_services" {}
`
