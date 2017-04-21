package digitalocean

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDigitalOceanFloatingIP_importBasicRegion(t *testing.T) {
	resourceName := "digitalocean_floating_ip.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanFloatingIPConfig_region,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDigitalOceanFloatingIP_importBasicDroplet(t *testing.T) {
	resourceName := "digitalocean_floating_ip.foobar"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanFloatingIPConfig_droplet(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
