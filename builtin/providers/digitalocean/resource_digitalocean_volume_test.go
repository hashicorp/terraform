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

func TestAccDigitalOceanVolume_Basic(t *testing.T) {
	name := fmt.Sprintf("volume-%s", acctest.RandString(10))

	volume := godo.Volume{
		Name: name,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckDigitalOceanVolumeConfig_basic, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanVolumeExists("digitalocean_volume.foobar", &volume),
					resource.TestCheckResourceAttr(
						"digitalocean_volume.foobar", "name", name),
					resource.TestCheckResourceAttr(
						"digitalocean_volume.foobar", "size", "100"),
					resource.TestCheckResourceAttr(
						"digitalocean_volume.foobar", "region", "nyc1"),
					resource.TestCheckResourceAttr(
						"digitalocean_volume.foobar", "description", "peace makes plenty"),
				),
			},
		},
	})
}

const testAccCheckDigitalOceanVolumeConfig_basic = `
resource "digitalocean_volume" "foobar" {
	region      = "nyc1"
	name        = "%s"
	size        = 100
	description = "peace makes plenty"
}`

func testAccCheckDigitalOceanVolumeExists(rn string, volume *godo.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("not found: %s", rn)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no volume ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		got, _, err := client.Storage.GetVolume(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}
		if got.Name != volume.Name {
			return fmt.Errorf("wrong volume found, want %q got %q", volume.Name, got.Name)
		}
		// get the computed volume details
		*volume = *got
		return nil
	}
}

func testAccCheckDigitalOceanVolumeDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_volume" {
			continue
		}

		// Try to find the volume
		_, _, err := client.Storage.GetVolume(context.Background(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Volume still exists")
		}
	}

	return nil
}

func TestAccDigitalOceanVolume_Droplet(t *testing.T) {
	var (
		volume  = godo.Volume{Name: fmt.Sprintf("volume-%s", acctest.RandString(10))}
		droplet godo.Droplet
	)
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanVolumeConfig_droplet(rInt, volume.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanVolumeExists("digitalocean_volume.foobar", &volume),
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					// the droplet should see an attached volume
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "volume_ids.#", "1"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanVolumeConfig_droplet(rInt int, vName string) string {
	return fmt.Sprintf(`
resource "digitalocean_volume" "foobar" {
	region      = "nyc1"
	name        = "%s"
	size        = 100
	description = "peace makes plenty"
}

resource "digitalocean_droplet" "foobar" {
  name               = "baz-%d"
  size               = "1gb"
  image              = "centos-7-x64"
  region             = "nyc1"
  ipv6               = true
  private_networking = true
  volume_ids         = ["${digitalocean_volume.foobar.id}"]
}`, vName, rInt)
}
