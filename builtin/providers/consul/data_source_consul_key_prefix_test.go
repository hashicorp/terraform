package consul

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataConsulKeyPrefix_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataConsulKeysConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulKeyPrefixValue("data.consul_key_prefix.read", "read", "written"),
				),
			},
		},
	})
}

const testAccDataConsulKeyPrefixConfig = `
resource "consul_key_prefix" "write" {
    datacenter = "dc1"

	path_prefix = "services/api/"
	subkeys = {
		mysql_hostname = "1.2.3.4"
		mysql_port = "3306"
	}
}

data "consul_key_prefix" "read" {
    # Create a dependency on the resource so we're sure to
    # have the value in place before we try to read it.
    datacenter = "${consul_keys.write.datacenter}"

	path_prefix = "services/api/"
}
`
