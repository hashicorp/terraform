package digitalocean

import (
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
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
			resource.TestStep{
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

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanFloatingIPConfig_droplet,
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
		_, _, err := client.FloatingIPs.Get(rs.Primary.ID)

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
		foundFloatingIP, _, err := client.FloatingIPs.Get(rs.Primary.ID)

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

var testAccCheckDigitalOceanFloatingIPConfig_droplet = fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name               = "baz"
  size               = "1gb"
  image              = "centos-5-8-x32"
  region             = "nyc3"
  ipv6               = true
  private_networking = true
  ssh_keys           = ["${digitalocean_ssh_key.foobar.id}"]
}

resource "digitalocean_floating_ip" "foobar" {
  droplet_id = "${digitalocean_droplet.foobar.id}"
  region     = "${digitalocean_droplet.foobar.region}"
}`, testAccValidPublicKey)
