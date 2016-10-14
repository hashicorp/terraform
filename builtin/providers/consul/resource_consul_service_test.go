package consul

import (
	"fmt"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccConsulService_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() {},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConsulServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccConsulServiceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulServiceExists(),
					testAccCheckConsulServiceValue("consul_service.app", "address", "www.google.com"),
					testAccCheckConsulServiceValue("consul_service.app", "id", "google"),
					testAccCheckConsulServiceValue("consul_service.app", "service_id", "google"),
					testAccCheckConsulServiceValue("consul_service.app", "name", "google"),
					testAccCheckConsulServiceValue("consul_service.app", "port", "80"),
					testAccCheckConsulServiceValue("consul_service.app", "tags.#", "2"),
					testAccCheckConsulServiceValue("consul_service.app", "tags.0", "tag0"),
					testAccCheckConsulServiceValue("consul_service.app", "tags.1", "tag1"),
				),
			},
		},
	})
}

func testAccCheckConsulServiceDestroy(s *terraform.State) error {
	agent := testAccProvider.Meta().(*consulapi.Client).Agent()
	services, err := agent.Services()
	if err != nil {
		return fmt.Errorf("Could not retrieve services: %#v", err)
	}
	_, ok := services["google"]
	if ok {
		return fmt.Errorf("Service still exists: %#v", "google")
	}
	return nil
}

func testAccCheckConsulServiceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		agent := testAccProvider.Meta().(*consulapi.Client).Agent()
		services, err := agent.Services()
		if err != nil {
			return err
		}
		_, ok := services["google"]
		if !ok {
			return fmt.Errorf("Service does not exist: %#v", "google")
		}
		return nil
	}
}

func testAccCheckConsulServiceValue(n, attr, val string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rn, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		out, ok := rn.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("Attribute '%s' not found: %#v", attr, rn.Primary.Attributes)
		}
		if val != "<any>" && out != val {
			return fmt.Errorf("Attribute '%s' value '%s' != '%s'", attr, out, val)
		}
		if val == "<any>" && out == "" {
			return fmt.Errorf("Attribute '%s' value '%s'", attr, out)
		}
		return nil
	}
}

const testAccConsulServiceConfig = `
resource "consul_service" "app" {
	address = "www.google.com"
	service_id = "google"
	name = "google"
	port = 80
	tags = ["tag0", "tag1"]
}
`
