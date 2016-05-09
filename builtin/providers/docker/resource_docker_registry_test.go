package docker

import (
	"fmt"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerRegistry_file(t *testing.T) {
	var ac dc.AuthConfigurations
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerRegistryFile,
				Check: resource.ComposeTestCheckFunc(
					testDockerAuthConfiguration("docker_registry.foo", &ac),
				),
			},
		},
	})
}

func TestAccDockerRegistry_config(t *testing.T) {
	var ac dc.AuthConfigurations
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAddDockerRegistryConfig,
				Check: resource.ComposeTestCheckFunc(
					testDockerAuthConfiguration("docker_registry.foobar", &ac),
				),
			},
		},
	})
}

func testDockerAuthConfiguration(n string, authConfigurations *dc.AuthConfigurations) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		return nil
	}
}

const testAccDockerRegistryFile = `
resource "docker_registry" "foo" {
  settings_file = "./test-fixtures/config.json"
}
`

const testAddDockerRegistryConfig = `
resource "docker_registry" "foobar" {
  auth = {
    username = "foo"
    password = "bar"
    server_address = "docker.io"
  }
}
`
