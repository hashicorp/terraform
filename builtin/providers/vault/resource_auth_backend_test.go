package vault

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccVaultAuthBackend_basic(t *testing.T) {
	var auth api.AuthMount
	path := fmt.Sprintf("path-%s/auth-%s",
		acctest.RandString(5), acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultAuthBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultAuthBackendConfig("app-id", path, "hello auth"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultAuthBackendExists("vault_auth_backend.foo", &auth),
					testAccCheckVaultAuthBackendAttributes(&auth, "app-id", "hello auth"),
				),
			},
		},
	})
}

func TestAccVaultAuthBackend_disappears(t *testing.T) {
	var auth api.AuthMount
	path := fmt.Sprintf("path-%s/auth-%s",
		acctest.RandString(5), acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultAuthBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultAuthBackendConfig("app-id", path, "hello auth"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultAuthBackendExists("vault_auth_backend.foo", &auth),
					testAccVaultAuthBackendDisappear(path),
				),
				ExpectNonEmptyPlan: true,
			},
			// Follow up plan w/ empty config should be empty, since the mount is
			// gone.
			resource.TestStep{
				Config: "",
			},
		},
	})
}

func TestAccVaultAuthBackend_implicitParams(t *testing.T) {
	var auth api.AuthMount

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultAuthBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultAuthBackendConfigMinimal("app-id"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultAuthBackendExists("vault_auth_backend.foo", &auth),
					testAccCheckVaultAuthBackendAttributes(&auth, "app-id", "Managed by Terraform"),
					resource.TestCheckResourceAttr("vault_auth_backend.foo", "path", "app-id"),
				),
			},
		},
	})
}

func testAccCheckVaultAuthBackendExists(
	key string, auth *api.AuthMount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[key]
		client := testAccProvider.Meta().(*api.Client)

		auths, err := client.Sys().ListAuth()
		if err != nil {
			return fmt.Errorf("Error listing mounts: %s", err)
		}

		// Paths from the API include an extra trailing slash
		a, ok := auths[fmt.Sprintf("%s/", rs.Primary.ID)]
		if !ok {
			return fmt.Errorf("Auth backend not found: %s", rs.Primary.ID)
		}

		*auth = *a
		return nil
	}
}

func testAccCheckVaultAuthBackendAttributes(
	auth *api.AuthMount,
	expectedType, expectedDescrip string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if auth.Type != expectedType {
			return fmt.Errorf("Expected auth type %q, got %q",
				expectedType, auth.Type)
		}
		if auth.Description != expectedDescrip {
			return fmt.Errorf("Expected auth description %q, got %q",
				expectedDescrip, auth.Description)
		}
		return nil
	}
}

func testAccCheckVaultAuthBackendDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*api.Client)

	existingAuthBackends, err := client.Sys().ListAuth()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_auth_backend" {
			continue
		}
		for mountPoint := range existingAuthBackends {
			if mountPoint == rs.Primary.ID {
				return fmt.Errorf("AuthBackend still exists: %s", mountPoint)
			}
		}
	}

	return nil
}

func testAccVaultAuthBackendDisappear(path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		return client.Sys().DisableAuth(path)
	}
}

func testAccVaultAuthBackendConfig(authType, path, descrip string) string {
	return fmt.Sprintf(`
resource "vault_auth_backend" "foo" {
  type              = "%s"
  path              = "%s"
  description       = "%s"
}
`, authType, path, descrip)
}

func testAccVaultAuthBackendConfigMinimal(authType string) string {
	return fmt.Sprintf(`
resource "vault_auth_backend" "foo" {
  type = "%s"
}
`, authType)
}
