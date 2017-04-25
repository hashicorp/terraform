package docker

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

var registryDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerRegistryImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_image.foo", "sha256_digest", registryDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerRegistryImage_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageDataSourcePrivateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_image.bar", "sha256_digest", registryDigestRegexp),
				),
			},
		},
	})
}

const testAccDockerImageDataSourceConfig = `
data "docker_registry_image" "foo" {
	name = "alpine:latest"
}
`

const testAccDockerImageDataSourcePrivateConfig = `
data "docker_registry_image" "bar" {
	name = "gcr.io:443/google_containers/pause:0.8.0"
}
`
