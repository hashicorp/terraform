package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccRoute53Zone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53ZoneConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ZoneExists("aws_route53_zone.main"),
				),
			},
		},
	})
}

func testAccCheckRoute53ZoneDestroy(s *terraform.State) error {
	conn := testAccProvider.route53
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_zone" {
			continue
		}

		_, err := conn.GetHostedZone(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Hosted zone still exists")
		}
	}
	return nil
}

func testAccCheckRoute53ZoneExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No hosted zone ID is set")
		}

		conn := testAccProvider.route53
		_, err := conn.GetHostedZone(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Hosted zone err: %v", err)
		}
		return nil
	}
}

const testAccRoute53ZoneConfig = `
resource "aws_route53_zone" "main" {
	name = "hashicorp.com"
}
`
