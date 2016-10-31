package rabbitmq

import (
	"fmt"
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccVhost(t *testing.T) {
	var vhost string
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccVhostCheckDestroy(vhost),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVhostConfig_basic,
				Check: testAccVhostCheck(
					"rabbitmq_vhost.test", &vhost,
				),
			},
		},
	})
}

func testAccVhostCheck(rn string, name *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("vhost id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		vhosts, err := rmqc.ListVhosts()
		if err != nil {
			return fmt.Errorf("Error retrieving vhosts: %s", err)
		}

		for _, vhost := range vhosts {
			if vhost.Name == rs.Primary.ID {
				*name = rs.Primary.ID
				return nil
			}
		}

		return fmt.Errorf("Unable to find vhost %s", rn)
	}
}

func testAccVhostCheckDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		vhosts, err := rmqc.ListVhosts()
		if err != nil {
			return fmt.Errorf("Error retrieving vhosts: %s", err)
		}

		for _, vhost := range vhosts {
			if vhost.Name == name {
				return fmt.Errorf("vhost still exists: %v", vhost)
			}
		}

		return nil
	}
}

const testAccVhostConfig_basic = `
resource "rabbitmq_vhost" "test" {
    name = "test"
}`
