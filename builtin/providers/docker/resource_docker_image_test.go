package docker

import (
	"fmt"
	"regexp"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var contentDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy,
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy,
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

func TestAccDockerImage_destroy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			for _, rs := range s.RootModule().Resources {
				if rs.Type != "docker_image" {
					continue
				}

				client := testAccProvider.Meta().(*dc.Client)
				_, err := client.InspectImage(rs.Primary.Attributes["latest"])
				if err != nil {
					return err
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageKeepLocallyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobarzoo", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_data(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageFromDataConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobarbaz", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_data_pull_trigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerImageFromDataConfigWithPullTrigger,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobarbazoo", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func testAccDockerImageDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "docker_image" {
			continue
		}

		client := testAccProvider.Meta().(*dc.Client)
		_, err := client.InspectImage(rs.Primary.Attributes["latest"])
		if err == nil {
			return fmt.Errorf("Image still exists")
		} else if err != dc.ErrNoSuchImage {
			return err
		}
	}
	return nil
}

const testAccDockerImageConfig = `
resource "docker_image" "foo" {
	name = "alpine:3.1"
}
`

const testAddDockerPrivateImageConfig = `
resource "docker_image" "foobar" {
	name = "gcr.io:443/google_containers/pause:0.8.0"
}
`

const testAccDockerImageKeepLocallyConfig = `
resource "docker_image" "foobarzoo" {
	name = "crux:3.1"
	keep_locally = true
}
`

const testAccDockerImageFromDataConfig = `
data "docker_registry_image" "foobarbaz" {
	name = "alpine:3.1"
}
resource "docker_image" "foobarbaz" {
	name = "${data.docker_registry_image.foobarbaz.name}"
	pull_triggers = ["${data.docker_registry_image.foobarbaz.sha256_digest}"]
}
`

const testAccDockerImageFromDataConfigWithPullTrigger = `
data "docker_registry_image" "foobarbazoo" {
	name = "alpine:3.1"
}
resource "docker_image" "foobarbazoo" {
	name = "${data.docker_registry_image.foobarbazoo.name}"
	pull_trigger = "${data.docker_registry_image.foobarbazoo.sha256_digest}"
}
`
