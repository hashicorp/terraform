package docker

import (
	"fmt"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerNetwork_basic(t *testing.T) {
	var n dc.Network

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
				),
			},
		},
	})
}

func testAccNetwork(n string, network *dc.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*dc.Client)
		networks, err := client.ListNetworks()
		if err != nil {
			return err
		}

		for _, n := range networks {
			if n.ID == rs.Primary.ID {
				inspected, err := client.NetworkInfo(n.ID)
				if err != nil {
					return fmt.Errorf("Network could not be obtained: %s", err)
				}
				*network = *inspected
				return nil
			}
		}

		return fmt.Errorf("Network not found: %s", rs.Primary.ID)
	}
}

const testAccDockerNetworkConfig = `
resource "docker_network" "foo" {
  name = "bar"
}
`

func TestAccDockerNetwork_internal(t *testing.T) {
	var n dc.Network

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerNetworkInternalConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foobar", &n),
					testAccNetworkInternal(&n, true),
				),
			},
		},
	})
}

func testAccNetworkInternal(network *dc.Network, internal bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Internal != internal {
			return fmt.Errorf("Bad value for attribute 'internal': %t", network.Internal)
		}
		return nil
	}
}

const testAccDockerNetworkInternalConfig = `
resource "docker_network" "foobar" {
  name = "foobar"
  internal = "true"
}
`
