package openstack

import (
	"fmt"
	"testing"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
)

func TestAccOpenstackNetwork(t *testing.T) {
	var network network.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpenstackNetworkExists("openstack_network.accept_test", &network),
					resource.TestCheckResourceAttr(
						"openstack_network.accept_test", "name", "accept_test Network"),
				),
			},
		},
	})
}

func testAccCheckOpenstackNetworkDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "openstack_network" {
			continue
		}

		networksApi, err := getNetworkApi(client.AccessProvider)
		if err != nil {
			return err
		}

		_, err = networksApi.GetNetwork(rs.ID)
		if err == nil {
			return fmt.Errorf("Network (%s) still exists.", rs.ID)
		}

		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Unkonw network error")
		}

		if httpError.Actual == 404 {
			return nil
		}

		return err
	}

	return nil
}

func testAccCheckOpenstackNetworkExists(n string, network *network.Network) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		client := testAccProvider.client
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No network is set")
		}

		networksApi, err := getNetworkApi(client.AccessProvider)
		if err != nil {
			return err
		}

		network, err = networksApi.GetNetwork(rs.ID)
		if err != nil {
			return err
		}

		if len(network.Subnets) == 2 {
			return nil
		} else {
			return fmt.Errorf("Subnets not found")
		}
	}
}

const testNetworkConfig = `
resource "openstack_network" "accept_test" {
    name = "accept_test Network"
}
`
