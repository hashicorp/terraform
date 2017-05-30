package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceGoogleSubnetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: TestAccDataSourceGoogleSubnetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGoogleSubnetworkCheck("data.google_compute_subnetwork.my_subnetwork", "google_compute_subnetwork.foobar"),
				),
			},
		},
	})
}

func testAccDataSourceGoogleSubnetworkCheck(data_source_name string, resource_name string) resource.TestCheckFunc {
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

		subnetwork_attrs_to_test := []string{
			"id",
			"self_link",
			"name",
			"description",
			"ip_cidr_range",
			"network",
			"private_ip_google_access",
		}

		for _, attr_to_check := range subnetwork_attrs_to_test {
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

var TestAccDataSourceGoogleSubnetworkConfig = `

resource "google_compute_network" "foobar" {
	name = "network-test"
	description = "my-description"
}
resource "google_compute_subnetwork" "foobar" {
	name = "subnetwork-test"
	description = "my-description"
	ip_cidr_range = "10.0.0.0/24"
	network  = "${google_compute_network.foobar.self_link}"
	private_ip_google_access = true
}

data "google_compute_subnetwork" "my_subnetwork" {
	name = "${google_compute_subnetwork.foobar.name}"
}
`
