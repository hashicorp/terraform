package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func TestAccOpenstackCompute(t *testing.T) {
	var server gophercloud.NewServer

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackComputeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testComputeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpenstackComputeExists("openstack_compute.accept_test", &server),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "name", "compute_instance"),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "image_ref", "a1e03b6d-2532-4f35-b6dc-761a087cf43e"),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "flavor_ref", "1"),
				),
			},
		},
	})
}

func testAccCheckOpenstackComputeDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "openstack_compute" {
			continue
		}

		serversApi, err := gophercloud.ServersApi(client.AccessProvider, gophercloud.ApiCriteria{
			Name:      "nova",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = serversApi.ServerById(rs.ID)
		if err == nil {
			return fmt.Errorf("Instance (%s) still exists.", rs.ID)
		}

		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Unkonw computeerror")
		}

		if httpError.Actual == 404 {
			return nil
		}

		return err
	}

	return nil
}

func testAccCheckOpenstackComputeExists(n string, server *gophercloud.NewServer) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		client := testAccProvider.client
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No network is set")
		}

		serversApi, err := gophercloud.ServersApi(client.AccessProvider, gophercloud.ApiCriteria{
			Name:      "nova",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = serversApi.ServerById(rs.ID)
		return err
	}
}

const testComputeConfig = `
resource "openstack_compute" "accept_test" {
    name = "accept_test_compute"
    image_ref = "a1e03b6d-2532-4f35-b6dc-761a087cf43e"
    flavor_ref = "1"
    count = 1
}
`
