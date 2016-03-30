package dockercloud

import (
	"fmt"
	"testing"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCheckDockercloudService_Basic(t *testing.T) {
	var service dockercloud.Service

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDockercloudServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDockercloudServiceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDockercloudServiceExists("dockercloud_service.foobar", &service),
					testAccCheckDockercloudServiceAttributes(&service),
					resource.TestCheckResourceAttr(
						"dockercloud_service.foobar", "name", "foobar-test-terraform"),
					resource.TestCheckResourceAttr(
						"dockercloud_service.foobar", "image", "python:3.2"),
					resource.TestCheckResourceAttr(
						"dockercloud_service.foobar", "entrypoint", "python -m http.server"),
				),
			},
		},
	})
}

func testAccCheckDockercloudServiceDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dockercloud_service" {
			continue
		}

		service, err := dockercloud.GetService(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving service: %s", err)
		}

		if service.State != "Terminated" {
			return fmt.Errorf("Service still running")
		}
	}

	return nil
}

func testAccCheckDockercloudServiceExists(n string, service *dockercloud.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		retrieveService, err := dockercloud.GetService(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving service: %s", err)
		}

		if retrieveService.Uuid != rs.Primary.ID {
			return fmt.Errorf("Service not found")
		}

		*service = retrieveService

		return nil
	}
}

func testAccCheckDockercloudServiceAttributes(service *dockercloud.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != "foobar-test-terraform" {
			return fmt.Errorf("Bad name: %s", service.Name)
		}

		if service.Image_name != "python:3.2" {
			return fmt.Errorf("Bad image: %s", service.Image_name)
		}

		if service.Entrypoint != "python -m http.server" {
			return fmt.Errorf("Bad entrypoint: %s", service.Entrypoint)
		}

		return nil
	}
}

const testAccCheckDockercloudServiceConfig_basic = `
resource "dockercloud_node_cluster" "foobar" {
    name = "foobar-test-terraform"
    node_provider = "aws"
    size = "t2.micro"
    region = "us-east-1"
}

resource "dockercloud_service" "foobar" {
    name = "foobar-test-terraform"
    image = "python:3.2"
    entrypoint = "python -m http.server"

    depends_on = ["dockercloud_node_cluster.foobar"]
}`
