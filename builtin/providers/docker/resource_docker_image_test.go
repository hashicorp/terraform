package docker

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
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
						"b7cf8f0d9e82c9d96bd7afd22c600bfdb86b8d66c50d29164e5ad2fb02f7187b"),
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
	name = "ubuntu:trusty-20150320"
	keep_updated = true
}
`

const testAddDockerPrivateImageConfig = `
resource "docker_image" "foobar" {
	name = "gcr.io:443/google_containers/pause:0.8.0"
	keep_updated = true
}
`
