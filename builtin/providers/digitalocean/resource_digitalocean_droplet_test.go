package digitalocean

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanDroplet_Basic(t *testing.T) {
	var droplet godo.Droplet

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
						"digitalocean_droplet.foobar", "image", "centos-7-x64"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "region", "nyc3"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "user_data", "foobar"),
				),
			},
		},
	})
}

func TestAccDigitalOceanDroplet_Update(t *testing.T) {
	var droplet godo.Droplet

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
				),
			},

			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_RenameAndResize,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					testAccCheckDigitalOceanDropletRenamedAndResized(&droplet),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "name", "baz"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "size", "1gb"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "disk", "30"),
				),
			},
		},
	})
}

func TestAccDigitalOceanDroplet_ResizeWithOutDisk(t *testing.T) {
	var droplet godo.Droplet

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
				),
			},

			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_resize_without_disk,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					testAccCheckDigitalOceanDropletResizeWithOutDisk(&droplet),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "size", "1gb"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "disk", "20"),
				),
			},
		},
	})
}

func TestAccDigitalOceanDroplet_UpdateUserData(t *testing.T) {
	var afterCreate, afterUpdate godo.Droplet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &afterCreate),
					testAccCheckDigitalOceanDropletAttributes(&afterCreate),
				),
			},

			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_userdata_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &afterUpdate),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar",
						"user_data",
						"foobar foobar"),
					testAccCheckDigitalOceanDropletRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func TestAccDigitalOceanDroplet_UpdateTags(t *testing.T) {
	var afterCreate, afterUpdate godo.Droplet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &afterCreate),
					testAccCheckDigitalOceanDropletAttributes(&afterCreate),
				),
			},

			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_tag_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &afterUpdate),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar",
						"tags.#",
						"1"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar",
						"tags.0",
						"barbaz"),
				),
			},
		},
	})
}

func TestAccDigitalOceanDroplet_PrivateNetworkingIpv6(t *testing.T) {
	var droplet godo.Droplet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanDropletConfig_PrivateNetworkingIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					testAccCheckDigitalOceanDropletAttributes_PrivateNetworkingIpv6(&droplet),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "private_networking", "true"),
					resource.TestCheckResourceAttr(
						"digitalocean_droplet.foobar", "ipv6", "true"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanDropletDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_droplet" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		// Try to find the Droplet
		_, _, err = client.Droplets.Get(id)

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for droplet (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckDigitalOceanDropletAttributes(droplet *godo.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if droplet.Image.Slug != "centos-7-x64" {
			return fmt.Errorf("Bad image_slug: %s", droplet.Image.Slug)
		}

		if droplet.Size.Slug != "512mb" {
			return fmt.Errorf("Bad size_slug: %s", droplet.Size.Slug)
		}

		if droplet.Region.Slug != "nyc3" {
			return fmt.Errorf("Bad region_slug: %s", droplet.Region.Slug)
		}

		if droplet.Name != "foo" {
			return fmt.Errorf("Bad name: %s", droplet.Name)
		}
		return nil
	}
}

func testAccCheckDigitalOceanDropletRenamedAndResized(droplet *godo.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if droplet.Size.Slug != "1gb" {
			return fmt.Errorf("Bad size_slug: %s", droplet.SizeSlug)
		}

		if droplet.Name != "baz" {
			return fmt.Errorf("Bad name: %s", droplet.Name)
		}

		if droplet.Disk != 30 {
			return fmt.Errorf("Bad disk: %d", droplet.Disk)
		}

		return nil
	}
}

func testAccCheckDigitalOceanDropletResizeWithOutDisk(droplet *godo.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if droplet.Size.Slug != "1gb" {
			return fmt.Errorf("Bad size_slug: %s", droplet.SizeSlug)
		}

		if droplet.Disk != 20 {
			return fmt.Errorf("Bad disk: %d", droplet.Disk)
		}

		return nil
	}
}

func testAccCheckDigitalOceanDropletAttributes_PrivateNetworkingIpv6(droplet *godo.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if droplet.Image.Slug != "centos-7-x64" {
			return fmt.Errorf("Bad image_slug: %s", droplet.Image.Slug)
		}

		if droplet.Size.Slug != "1gb" {
			return fmt.Errorf("Bad size_slug: %s", droplet.Size.Slug)
		}

		if droplet.Region.Slug != "sgp1" {
			return fmt.Errorf("Bad region_slug: %s", droplet.Region.Slug)
		}

		if droplet.Name != "baz" {
			return fmt.Errorf("Bad name: %s", droplet.Name)
		}

		if findIPv4AddrByType(droplet, "private") == "" {
			return fmt.Errorf("No ipv4 private: %s", findIPv4AddrByType(droplet, "private"))
		}

		// if droplet.IPV6Address("private") == "" {
		// 	return fmt.Errorf("No ipv6 private: %s", droplet.IPV6Address("private"))
		// }

		if findIPv4AddrByType(droplet, "public") == "" {
			return fmt.Errorf("No ipv4 public: %s", findIPv4AddrByType(droplet, "public"))
		}

		if findIPv6AddrByType(droplet, "public") == "" {
			return fmt.Errorf("No ipv6 public: %s", findIPv6AddrByType(droplet, "public"))
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "digitalocean_droplet" {
				continue
			}
			if rs.Primary.Attributes["ipv6_address"] != strings.ToLower(findIPv6AddrByType(droplet, "public")) {
				return fmt.Errorf("IPV6 Address should be lowercase")
			}

		}

		return nil
	}
}

func testAccCheckDigitalOceanDropletExists(n string, droplet *godo.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Droplet ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		// Try to find the Droplet
		retrieveDroplet, _, err := client.Droplets.Get(id)

		if err != nil {
			return err
		}

		if strconv.Itoa(retrieveDroplet.ID) != rs.Primary.ID {
			return fmt.Errorf("Droplet not found")
		}

		*droplet = *retrieveDroplet

		return nil
	}
}

func testAccCheckDigitalOceanDropletRecreated(t *testing.T,
	before, after *godo.Droplet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if before.ID == after.ID {
			t.Fatalf("Expected change of droplet IDs, but both were %v", before.ID)
		}
		return nil
	}
}

var testAccCheckDigitalOceanDropletConfig_basic = fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name      = "foo"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
  user_data = "foobar"
  ssh_keys  = ["${digitalocean_ssh_key.foobar.id}"]
}
`, testAccValidPublicKey)

var testAccCheckDigitalOceanDropletConfig_tag_update = fmt.Sprintf(`
resource "digitalocean_tag" "barbaz" {
  name       = "barbaz"
}

resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name      = "foo"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
  user_data = "foobar"
  ssh_keys  = ["${digitalocean_ssh_key.foobar.id}"]
  tags  = ["${digitalocean_tag.barbaz.id}"]
}
`, testAccValidPublicKey)

var testAccCheckDigitalOceanDropletConfig_userdata_update = fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name      = "foo"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
  user_data = "foobar foobar"
  ssh_keys  = ["${digitalocean_ssh_key.foobar.id}"]
}
`, testAccValidPublicKey)

var testAccCheckDigitalOceanDropletConfig_RenameAndResize = fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name     = "baz"
  size     = "1gb"
  image    = "centos-7-x64"
  region   = "nyc3"
  ssh_keys = ["${digitalocean_ssh_key.foobar.id}"]
}
`, testAccValidPublicKey)

var testAccCheckDigitalOceanDropletConfig_resize_without_disk = fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name     = "foo"
  size     = "1gb"
  image    = "centos-7-x64"
  region   = "nyc3"
  user_data = "foobar"
  ssh_keys = ["${digitalocean_ssh_key.foobar.id}"]
  resize_disk = false
}
`, testAccValidPublicKey)

// IPV6 only in singapore
var testAccCheckDigitalOceanDropletConfig_PrivateNetworkingIpv6 = fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
  name       = "foobar"
  public_key = "%s"
}

resource "digitalocean_droplet" "foobar" {
  name               = "baz"
  size               = "1gb"
  image              = "centos-7-x64"
  region             = "sgp1"
  ipv6               = true
  private_networking = true
  ssh_keys           = ["${digitalocean_ssh_key.foobar.id}"]
}
`, testAccValidPublicKey)
