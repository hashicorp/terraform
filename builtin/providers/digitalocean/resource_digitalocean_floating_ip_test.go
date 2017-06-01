package digitalocean

import (
	"context"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanFloatingIP_Region(t *testing.T) {
	var floatingIP godo.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanFloatingIPConfig_region,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanFloatingIPExists("digitalocean_floating_ip.foobar", &floatingIP),
					resource.TestCheckResourceAttr(
						"digitalocean_floating_ip.foobar", "region", "nyc3"),
				),
			},
		},
	})
}

func TestAccDigitalOceanFloatingIP_Droplet(t *testing.T) {
	var floatingIP godo.FloatingIP
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanFloatingIPConfig_droplet(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanFloatingIPExists("digitalocean_floating_ip.foobar", &floatingIP),
					resource.TestCheckResourceAttr(
						"digitalocean_floating_ip.foobar", "region", "nyc3"),
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

		// Try to find the key
		_, _, err := client.FloatingIPs.Get(context.Background(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Floating IP still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanFloatingIPExists(n string, floatingIP *godo.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		// Try to find the FloatingIP
		foundFloatingIP, _, err := client.FloatingIPs.Get(context.Background(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundFloatingIP.IP != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*floatingIP = *foundFloatingIP

		return nil
	}
}

var testAccCheckDigitalOceanFloatingIPConfig_region = `
resource "digitalocean_floating_ip" "foobar" {
  region = "nyc3"
}`

func testAccCheckDigitalOceanFloatingIPConfig_droplet(rInt int) string {
	return fmt.Sprintf(`
resource "digitalocean_droplet" "foobar" {
  name               = "baz-%d"
  size               = "1gb"
  image              = "centos-7-x64"
  region             = "nyc3"
  ipv6               = true
  private_networking = true
}

resource "digitalocean_floating_ip" "foobar" {
  droplet_id = "${digitalocean_droplet.foobar.id}"
  region     = "${digitalocean_droplet.foobar.region}"
}`, rInt)
}
