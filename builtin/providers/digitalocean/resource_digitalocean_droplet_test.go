package digitalocean

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

func TestAccDigitalOceanDroplet(t *testing.T) {
	var droplet digitalocean.Droplet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					testAccCheckDigitalOceanDropletAttributes(&droplet),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "name", "foo"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "size", "512mb"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "image", "centos-5-8-x32"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "region", "nyc2"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanDropletDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "digitalocean_droplet" {
			continue
		}

		// Try to find the Droplet
		_, err := client.RetrieveDroplet(rs.ID)

		stateConf := &resource.StateChangeConf{
			Pending: []string{"new"},
			Target:  "off",
			Refresh: DropletStateRefreshFunc(client, rs.ID),
			Timeout: 10 * time.Minute,
		}

		_, err = stateConf.WaitForState()

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for droplet (%s) to be destroyed: %s",
				rs.ID, err)
		}
	}

	return nil
}

func testAccCheckDigitalOceanDropletAttributes(droplet *digitalocean.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		fmt.Println("droplet: %v", droplet)
		if droplet.ImageSlug() != "centos-5-8-x32" {
			return fmt.Errorf("Bad image_slug: %s", droplet.ImageSlug())
		}

		if droplet.SizeSlug() != "512mb" {
			return fmt.Errorf("Bad size_slug: %s", droplet.SizeSlug())
		}

		if droplet.RegionSlug() != "nyc2" {
			return fmt.Errorf("Bad region_slug: %s", droplet.RegionSlug())
		}

		if droplet.Name != "foo" {
			return fmt.Errorf("Bad name: %s", droplet.Name)
		}
		return nil
	}
}

func testAccCheckDigitalOceanDropletExists(n string, droplet *digitalocean.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Droplet ID is set")
		}

		client := testAccProvider.client

		retrieveDroplet, err := client.RetrieveDroplet(rs.ID)

		if err != nil {
			return err
		}

		if retrieveDroplet.StringId() != rs.ID {
			return fmt.Errorf("Droplet not found")
		}

		*droplet = retrieveDroplet

		return nil
	}
}

const testAccCheckDigitalOceanDropletConfig_basic = `
resource "digitalocean_droplet" "foobar" {
    name = "foo"
    size = "512mb"
    image = "centos-5-8-x32"
    region = "nyc2"
}
`
