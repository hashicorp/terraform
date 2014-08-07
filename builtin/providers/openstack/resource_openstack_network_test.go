package openstack

import (
	"fmt"
	"testing"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func TestAccOpenstackNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpenstackNetworkExists("openstack_network.accept_test"),
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

		networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
			Name:      "neutron",
			UrlChoice: gophercloud.PublicURL,
		})
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

func testAccCheckOpenstackNetworkExists(n string) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		client := testAccProvider.client
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No network is set")
		}

		networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
			Name:      "neutron",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = networksApi.GetNetwork(rs.ID)
		return err
	}
}

const testNetworkConfig = `
resource "openstack_network" "accept_test" {
    name = "accept_test Network"
}
`
