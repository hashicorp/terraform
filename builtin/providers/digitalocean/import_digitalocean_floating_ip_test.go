package digitalocean

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDigitalOceanFloatingIP_importBasicRegion(t *testing.T) {
	resourceName := "digitalocean_floating_ip.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanFloatingIPConfig_region,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDigitalOceanFloatingIP_importBasicDroplet(t *testing.T) {
	resourceName := "digitalocean_floating_ip.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanFloatingIPConfig_droplet,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
