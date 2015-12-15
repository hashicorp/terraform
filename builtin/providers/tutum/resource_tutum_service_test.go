package tutum

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/tutumcloud/go-tutum/tutum"
)

func TestAccCheckTutumService_Basic(t *testing.T) {
	var service tutum.Service

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTutumServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckTutumServiceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTutumServiceExists("tutum_service.foobar", &service),
					testAccCheckTutumServiceAttributes(&service),
					resource.TestCheckResourceAttr(
						"tutum_service.foobar", "name", "foobar-test-terraform"),
					resource.TestCheckResourceAttr(
						"tutum_service.foobar", "image", "python:3.2"),
					resource.TestCheckResourceAttr(
						"tutum_service.foobar", "entrypoint", "python -m http.server"),
				),
			},
		},
	})
}

func testAccCheckTutumServiceDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tutum_service" {
			continue
		}

		service, err := tutum.GetService(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving service: %s", err)
		}

		if service.State != "Terminated" {
			return fmt.Errorf("Service still running")
		}
	}

	return nil
}

func testAccCheckTutumServiceExists(n string, service *tutum.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		retrieveService, err := tutum.GetService(rs.Primary.ID)
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

func testAccCheckTutumServiceAttributes(service *tutum.Service) resource.TestCheckFunc {
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

const testAccCheckTutumServiceConfig_basic = `
resource "tutum_node_cluster" "foobar" {
    name = "foobar-test-terraform"
    node_provider = "aws"
    size = "t2.micro"
    region = "us-east-1"
}

resource "tutum_service" "foobar" {
    name = "foobar-test-terraform"
    image = "python:3.2"
    entrypoint = "python -m http.server"

    depends_on = ["tutum_node_cluster.foobar"]
}`
