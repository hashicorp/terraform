package rancher

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherRegistryCredential_basic(t *testing.T) {
	var registry rancherClient.RegistryCredential

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherRegistryCredentialDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherRegistryCredentialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistryCredentialExists("rancher_registry_credential.foo", &registry),
					resource.TestCheckResourceAttr("rancher_registry_credential.foo", "name", "foo"),
					resource.TestCheckResourceAttr("rancher_registry_credential.foo", "description", "registry credential test"),
					resource.TestCheckResourceAttr("rancher_registry_credential.foo", "public_value", "user"),
				),
			},
			resource.TestStep{
				Config: testAccRancherRegistryCredentialUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistryCredentialExists("rancher_registry_credential.foo", &registry),
					resource.TestCheckResourceAttr("rancher_registry_credential.foo", "name", "foo2"),
					resource.TestCheckResourceAttr("rancher_registry_credential.foo", "description", "registry credential test - updated"),
					resource.TestCheckResourceAttr("rancher_registry_credential.foo", "public_value", "user2"),
				),
			},
		},
	})
}

func TestAccRancherRegistryCredential_disappears(t *testing.T) {
	var registry rancherClient.RegistryCredential

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherRegistryCredentialDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherRegistryCredentialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistryCredentialExists("rancher_registry_credential.foo", &registry),
					testAccRancherRegistryCredentialDisappears(&registry),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRancherRegistryCredentialDisappears(reg *rancherClient.RegistryCredential) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(reg.AccountId)
		if err != nil {
			return err
		}

		// Step 1: Deactivate
		if _, e := client.RegistryCredential.ActionDeactivate(reg); e != nil {
			return fmt.Errorf("Error deactivating RegistryCredential: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"active", "inactive", "deactivating"},
			Target:     []string{"inactive"},
			Refresh:    RegistryCredentialStateRefreshFunc(client, reg.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, waitErr := stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for registry credential (%s) to be deactivated: %s", reg.Id, waitErr)
		}

		// Update resource to reflect its state
		reg, err = client.RegistryCredential.ById(reg.Id)
		if err != nil {
			return fmt.Errorf("Failed to refresh state of deactivated registry credential (%s): %s", reg.Id, err)
		}

		// Step 2: Remove
		if _, err := client.RegistryCredential.ActionRemove(reg); err != nil {
			return fmt.Errorf("Error removing RegistryCredential: %s", err)
		}

		stateConf = &resource.StateChangeConf{
			Pending:    []string{"inactive", "removed", "removing"},
			Target:     []string{"removed"},
			Refresh:    RegistryCredentialStateRefreshFunc(client, reg.Id),
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

func testAccCheckRancherRegistryCredentialExists(n string, reg *rancherClient.RegistryCredential) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client, err := testAccProvider.Meta().(*Config).RegistryClient(rs.Primary.Attributes["registry_id"])
		if err != nil {
			return err
		}

		foundReg, err := client.RegistryCredential.ById(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundReg.Resource.Id != rs.Primary.ID {
			return fmt.Errorf("RegistryCredential not found")
		}

		*reg = *foundReg

		return nil
	}
}

func testAccCheckRancherRegistryCredentialDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_registry_credential" {
			continue
		}
		client, err := testAccProvider.Meta().(*Config).GlobalClient()
		if err != nil {
			return err
		}

		reg, err := client.RegistryCredential.ById(rs.Primary.ID)

		if err == nil {
			if reg != nil &&
				reg.Resource.Id == rs.Primary.ID &&
				reg.State != "removed" {
				return fmt.Errorf("RegistryCredential still exists")
			}
		}

		return nil
	}
	return nil
}

const testAccRancherRegistryCredentialConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_registry" "foo" {
  name = "foo"
  description = "registry test"
  server_address = "http://bar.com:8080"
  environment_id = "${rancher_environment.foo.id}"
}

resource "rancher_registry_credential" "foo" {
	name = "foo"
	description = "registry credential test"
	registry_id = "${rancher_registry.foo.id}"
	email = "registry@credential.com"
	public_value = "user"
	secret_value = "pass"
}
`

const testAccRancherRegistryCredentialUpdateConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_registry" "foo" {
  name = "foo"
  description = "registry test"
  server_address = "http://bar.com:8080"
  environment_id = "${rancher_environment.foo.id}"
}

resource "rancher_registry_credential" "foo" {
	name = "foo2"
	description = "registry credential test - updated"
	registry_id = "${rancher_registry.foo.id}"
	email = "registry@credential.com"
	public_value = "user2"
	secret_value = "pass"
}
 `
