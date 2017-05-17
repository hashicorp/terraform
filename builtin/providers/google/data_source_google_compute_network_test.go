package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceGoogleNetwork(t *testing.T) {
	networkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceGoogleNetworkConfig(networkName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGoogleNetworkCheck("data.google_compute_network.my_network", "google_compute_network.foobar"),
				),
			},
		},
	})
}

func testAccDataSourceGoogleNetworkCheck(data_source_name string, resource_name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[data_source_name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", data_source_name)
		}

		rs, ok := s.RootModule().Resources[resource_name]
		if !ok {
			return fmt.Errorf("can't find %s in state", resource_name)
		}

		ds_attr := ds.Primary.Attributes
		rs_attr := rs.Primary.Attributes
		network_attrs_to_test := []string{
			"id",
			"self_link",
			"name",
			"description",
		}

		for _, attr_to_check := range network_attrs_to_test {
			if ds_attr[attr_to_check] != rs_attr[attr_to_check] {
				return fmt.Errorf(
					"%s is %s; want %s",
					attr_to_check,
					ds_attr[attr_to_check],
					rs_attr[attr_to_check],
				)
			}
		}
		return nil
	}
}

func testAccDataSourceGoogleNetworkConfig(name string) string {
	return fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "%s"
	description = "my-description"
}

data "google_compute_network" "my_network" {
	name = "${google_compute_network.foobar.name}"
}`, name)
}
