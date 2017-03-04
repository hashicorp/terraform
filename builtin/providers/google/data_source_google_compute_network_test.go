package google

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccDataSourceGoogleNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: TestAccDataSourceGoogleNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGoogleNetworktCheck("data.google_compute_network.my_network"),
				),
			},
		},
	})
}

func testAccDataSourceGoogleNetworktCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		networkOrigin, ok := s.RootModule().Resources["google_compute_network.foobar"]
		if !ok {
			return fmt.Errorf("can't find google_compute_network.foobar in state")
		}

		attr := rs.Primary.Attributes

		if attr["id"] != networkOrigin.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				networkOrigin.Primary.Attributes["id"],
			)
		}

		if attr["self_link"] != networkOrigin.Primary.Attributes["self_link"] {
			return fmt.Errorf(
				"self_link is %s; want %s",
				attr["self_link"],
				networkOrigin.Primary.Attributes["self_link"],
			)
		}

		if attr["name"] != "network-test" {
			return fmt.Errorf("bad name %s", attr["name"])
		}
		if attr["description"] != "my-description" {
			return fmt.Errorf("bad description %s", attr["description"])
		}

		return nil
	}
}

var TestAccDataSourceGoogleNetworkConfig = `
resource "google_compute_network" "foobar" {
	name = "network-test"
	description = "my-description"
}

data "google_compute_network" "my_network" {
	name = "${google_compute_network.foobar.name}"
}`
