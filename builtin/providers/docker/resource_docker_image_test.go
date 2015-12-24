package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDockerImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"docker_image.foo",
						"latest",
						"8dd8107abd2e22bfd3b45b05733f3d2677d4078b09b5edce56ee3d8677d3c648"),
				),
			},
		},
	})
}

func TestAddDockerImage_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAddDockerPrivateImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"docker_image.foobar",
						"latest",
						"2c40b0526b6358710fd09e7b8c022429268cc61703b4777e528ac9d469a07ca1"),
				),
			},
		},
	})
}

const testAccDockerImageConfig = `
resource "docker_image" "foo" {
	name = "alpine:3.1"
	keep_updated = false
}
`

const testAddDockerPrivateImageConfig = `
resource "docker_image" "foobar" {
	name = "gcr.io:443/google_containers/pause:0.8.0"
	keep_updated = true
}
`
