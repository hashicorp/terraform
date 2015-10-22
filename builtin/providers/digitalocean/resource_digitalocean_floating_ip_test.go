package digitalocean

import (
	"fmt"
	"testing"
	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanFloatingIP_Basic(t *testing.T) {
	var floatingIP godo.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanFloatingIPConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanFloatingIPExists("digitalocean_floating_ip.foobar", &floatingIP),
					testAccCheckDigitalOceanFloatingIPAttributes(&floatingIP),
					resource.TestCheckResourceAttr("digitalocean_floating_ip.foobar", "region", "ams3"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanFloatingIPDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_floating_ip" {
			continue
		}

		// Try to find the Floatin IP
		_, _, err := client.FloatingIPs.Get(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Floating IP still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanFloatingIPAttributes(floatingIP *godo.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if floatingIP.Region.Slug != "ams3" {
			return fmt.Errorf("Bad value: %s", floatingIP.Region.Slug)
		}

		return nil
	}
}

func testAccCheckDigitalOceanFloatingIPExists(n string, floatingIP *godo.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Floating IP ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		foundFloatingIP, _, err := client.FloatingIPs.Get(rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundFloatingIP.IP != rs.Primary.ID {
			return fmt.Errorf("Floating IP not found")
		}

		*floatingIP = *foundFloatingIP

		return nil
	}
}

const testAccCheckDigitalOceanFloatingIPConfig_basic = `
resource "digitalocean_floating_ip" "foobar" {
    region = "ams3"
}`
