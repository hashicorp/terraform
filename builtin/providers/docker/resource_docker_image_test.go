package docker

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

var contentDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAddDockerPrivateImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobar", "latest", contentDigestRegexp),
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
