package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAvailabilityZones_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsAvailabilityZonesDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsAvailabilityZonesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAvailabilityZonesID("aws_availability_zones.availability_zones"),
				),
			},
		},
	})
}

func testAccCheckAwsAvailabilityZonesDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAwsAvailabilityZonesID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find AZ resource: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("AZ resource ID not set")
		}
		return nil
	}
}

const testAccCheckAwsAvailabilityZonesConfig = `
resource "aws_availability_zones" "availability_zones" {
}
`
