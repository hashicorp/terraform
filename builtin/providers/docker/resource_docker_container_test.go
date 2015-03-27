package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDockerContainer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"docker_image.foo",
						"latest",
						"d0955f21bf24f5bfffd32d2d0bb669d0564701c271bc3dfc64cfc5adfdec2d07"),
				),
			},
		},
	})
}

const testAccDockerContainerConfig = `
resource "docker_image" "foo" {
	name = "ubuntu:trusty-20150320"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
}
`
