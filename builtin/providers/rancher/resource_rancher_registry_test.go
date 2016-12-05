package rancher

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherRegistry(t *testing.T) {
	var registry rancherClient.Registry

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherRegistryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherRegistryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistryExists("rancher_registry.foo", &registry),
					resource.TestCheckResourceAttr("rancher_registry.foo", "name", "foo"),
					resource.TestCheckResourceAttr("rancher_registry.foo", "description", "registry test"),
					resource.TestCheckResourceAttr("rancher_registry.foo", "server_address", "http://foo.com:8080"),
				),
			},
			resource.TestStep{
				Config: testAccRancherRegistryUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistryExists("rancher_registry.foo", &registry),
					resource.TestCheckResourceAttr("rancher_registry.foo", "name", "foo2"),
					resource.TestCheckResourceAttr("rancher_registry.foo", "description", "registry test - updated"),
					resource.TestCheckResourceAttr("rancher_registry.foo", "server_address", "http://foo.updated.com:8080"),
				),
			},
			resource.TestStep{
				Config: testAccRancherRegistryRecreateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistryExists("rancher_registry.foo", &registry),
					resource.TestCheckResourceAttr("rancher_registry.foo", "name", "foo"),
					resource.TestCheckResourceAttr("rancher_registry.foo", "description", "registry test"),
					resource.TestCheckResourceAttr("rancher_registry.foo", "server_address", "http://foo.com:8080"),
				),
			},
		},
	})
}

func testAccCheckRancherRegistryExists(n string, reg *rancherClient.Registry) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*Config)

		foundReg, err := client.Registry.ById(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundReg.Resource.Id != rs.Primary.ID {
			return fmt.Errorf("Environment not found")
		}

		*reg = *foundReg

		return nil
	}
}

func testAccCheckRancherRegistryDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_registry" {
			continue
		}
		reg, err := client.Registry.ById(rs.Primary.ID)

		if err == nil {
			if reg != nil &&
				reg.Resource.Id == rs.Primary.ID &&
				reg.State != "removed" {
				return fmt.Errorf("Registry still exists")
			}
		}

		return nil
	}
	return nil
}

const testAccRancherRegistryConfig = `
resource "rancher_environment" "foo_registry" {
	name = "registry test"
	description = "environment to test registries"
}

resource "rancher_registry" "foo" {
  name = "foo"
  description = "registry test"
  server_address = "http://foo.com:8080"
  environment_id = "${rancher_environment.foo_registry.id}"
}
`

const testAccRancherRegistryUpdateConfig = `
 resource "rancher_environment" "foo_registry" {
   name = "registry test"
   description = "environment to test registries"
 }

 resource "rancher_registry" "foo" {
   name = "foo2"
   description = "registry test - updated"
   server_address = "http://foo.updated.com:8080"
   environment_id = "${rancher_environment.foo_registry.id}"
 }
 `

const testAccRancherRegistryRecreateConfig = `
 resource "rancher_environment" "foo_registry" {
   name = "registry test"
   description = "environment to test registries"
 }

 resource "rancher_environment" "foo_registry2" {
   name = "alternative registry test"
   description = "other environment to test registries"
 }

 resource "rancher_registry" "foo" {
   name = "foo"
   description = "registry test"
   server_address = "http://foo.com:8080"
   environment_id = "${rancher_environment.foo_registry2.id}"
 }
 `
