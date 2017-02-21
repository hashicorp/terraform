package rancher

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherRegistry_basic(t *testing.T) {
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

func TestAccRancherRegistry_disappears(t *testing.T) {
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
					testAccRancherRegistryDisappears(&registry),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRancherRegistryDisappears(reg *rancherClient.Registry) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(reg.AccountId)
		if err != nil {
			return err
		}

		// Step 1: Deactivate
		if _, e := client.Registry.ActionDeactivate(reg); e != nil {
			return fmt.Errorf("Error deactivating Registry: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"active", "inactive", "deactivating"},
			Target:     []string{"inactive"},
			Refresh:    RegistryStateRefreshFunc(client, reg.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, waitErr := stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for registry (%s) to be deactivated: %s", reg.Id, waitErr)
		}

		// Update resource to reflect its state
		reg, err = client.Registry.ById(reg.Id)
		if err != nil {
			return fmt.Errorf("Failed to refresh state of deactivated registry (%s): %s", reg.Id, err)
		}

		// Step 2: Remove
		if _, err := client.Registry.ActionRemove(reg); err != nil {
			return fmt.Errorf("Error removing Registry: %s", err)
		}

		stateConf = &resource.StateChangeConf{
			Pending:    []string{"inactive", "removed", "removing"},
			Target:     []string{"removed"},
			Refresh:    RegistryStateRefreshFunc(client, reg.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, waitErr = stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for registry (%s) to be removed: %s", reg.Id, waitErr)
		}

		return nil
	}
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

		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(rs.Primary.Attributes["environment_id"])
		if err != nil {
			return err
		}

		foundReg, err := client.Registry.ById(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundReg.Resource.Id != rs.Primary.ID {
			return fmt.Errorf("Registry not found")
		}

		*reg = *foundReg

		return nil
	}
}

func testAccCheckRancherRegistryDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_registry" {
			continue
		}
		client, err := testAccProvider.Meta().(*Config).GlobalClient()
		if err != nil {
			return err
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
