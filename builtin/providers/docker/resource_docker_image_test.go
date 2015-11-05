package docker

import (
	"testing"

	"fmt"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy(basicImageID),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"docker_image.foo",
						"latest",
						basicImageID),
				),
			},
		},
	})
}

func TestAddDockerImage_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy(privateImageID),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAddDockerPrivateImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"docker_image.foobar",
						"latest",
						privateImageID),
				),
			},
		},
	})
}

func testAccDockerImageDestroy(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*dc.Client)
		_, err := client.InspectImage(n)
		if err == nil {
			return fmt.Errorf("Image still exists")
		} else if err != dc.ErrNoSuchImage {
			return err
		}
		return nil
	}
}

const basicImageID = "f4fddc471ec22fc1f7d37768132f1753bc171121e30ac2af7fcb0302588197c0"

const testAccDockerImageConfig = `
resource "docker_image" "foo" {
	name = "alpine:3.2"
	keep_updated = true
}
`
const privateImageID = "2c40b0526b6358710fd09e7b8c022429268cc61703b4777e528ac9d469a07ca1"

const testAddDockerPrivateImageConfig = `
resource "docker_image" "foobar" {
	name = "gcr.io:443/google_containers/pause:0.8.0"
	keep_updated = true
}
`
