package consul

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataConsulKeyPrefix_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataConsulKeyPrefixConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulKeyPrefixExists("data.consul_key_prefix.read", "cheese", true),
					testAccCheckConsulKeyPrefixExists("data.consul_key_prefix.read", "bread", true),
					testAccCheckConsulKeyPrefixValue("data.consul_key_prefix.read", "cheese", "chevre"),
					testAccCheckConsulKeyPrefixValue("data.consul_key_prefix.read", "bread", "baguette"),
				),
			},
			resource.TestStep{
				Config: testAccDataConsulKeyPrefixConfig_Update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulKeyPrefixExists("data.consul_key_prefix.read", "cheese", false),
					testAccCheckConsulKeyPrefixExists("data.consul_key_prefix.read", "bread", true),
					testAccCheckConsulKeyPrefixExists("data.consul_key_prefix.read", "meat", true),
					testAccCheckConsulKeyPrefixValue("data.consul_key_prefix.read", "bread", "batard"),
					testAccCheckConsulKeyPrefixValue("data.consul_key_prefix.read", "meat", "ham"),
				),
			},
		},
	})
}

const testAccDataConsulKeyPrefixConfig = `
resource "consul_key_prefix" "app" {
        datacenter = "dc1"

    path_prefix = "prefix_test/"

    subkeys = {
        cheese = "chevre"
        bread = "baguette"
    }
}

data "consul_key_prefix" "read" {
    datacenter = "${consul_key_prefix.app.datacenter}"
    depends_on = ["consul_key_prefix.app"]

    path_prefix = "prefix_test/"
}
`

const testAccDataConsulKeyPrefixConfig_Update = `
resource "consul_key_prefix" "app" {
        datacenter = "dc1"

    path_prefix = "prefix_test/"

    subkeys = {
        bread = "batard"
        meat = "ham"
    }
}

data "consul_key_prefix" "read" {
    datacenter = "${consul_key_prefix.app.datacenter}"
    depends_on = ["consul_key_prefix.app"]

    path_prefix = "prefix_test/"
}
`

func testAccCheckConsulKeyPrefixValue(n string, attr string, val string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rn, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Data source not found")
		}
		out, ok := rn.Primary.Attributes["var."+attr]
		if !ok {
			return fmt.Errorf("Attribute '%s' not found: %#v", attr, rn.Primary.Attributes)
		}
		if val != "<any>" && out != val {
			return fmt.Errorf("Attribute '%s' value '%s' != '%s'", attr, out, val)
		}
		if val == "<any>" && out == "" {
			return fmt.Errorf("Attribute '%s' value '%s'", attr, out)
		}
		return nil
	}
}

func testAccCheckConsulKeyPrefixExists(n string, attr string, shouldExists bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rn, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Data source not found")
		}
		_, ok = rn.Primary.Attributes["var."+attr]
		if shouldExists && !ok {
			return fmt.Errorf("Attribute '%s' not found: %#v", attr, rn.Primary.Attributes)
		}
		if !shouldExists && ok {
			return fmt.Errorf("Attribute '%s' still present: %#v", attr, rn.Primary.Attributes)
		}
		return nil
	}
}
