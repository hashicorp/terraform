package docker

import (
	"fmt"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerContainer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo"),
				),
			},
		},
	})
}

func testAccContainerRunning(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*dc.Client)
		containers, err := client.ListContainers(dc.ListContainersOptions{})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

const testAccDockerContainerConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
}
`
