package vault

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccVaultSecretBackend_basic(t *testing.T) {
	var mount api.MountOutput
	path := fmt.Sprintf("path-%s/secret-%s",
		acctest.RandString(5), acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(path, "hello world", "30m", "100m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendAttributes(&mount, "generic", "hello world", "30m", "100m"),
					testAccCheckVaultSecretBackendConfigAttributes(path, "30m", "100m"),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "type", "generic"),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "path", path),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "default_lease_ttl", "30m0s"),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "max_lease_ttl", "1h40m0s"),
				),
			},
		},
	})
}

func TestAccVaultSecretBackend_disappears(t *testing.T) {
	var mount api.MountOutput
	path := fmt.Sprintf("path-%s/secret-%s",
		acctest.RandString(5), acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(path, "hello world", "30m", "100m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccVaultSecretBackendDisappear(path),
				),
				ExpectNonEmptyPlan: true,
			},
			// Plan w/ empty config should be empty, since the mount is gone.
			resource.TestStep{
				Config: "",
			},
		},
	})
}

func TestAccVaultSecretBackend_updateTTLs(t *testing.T) {
	var mount api.MountOutput
	path := fmt.Sprintf("path-%s/secret-%s",
		acctest.RandString(5), acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(path, "hello world", "30m", "100m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendAttributes(&mount, "generic", "hello world", "30m", "100m"),
					testAccCheckVaultSecretBackendConfigAttributes(path, "30m", "100m"),
				),
			},
			// Change both TTLs
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(path, "hello world", "60m", "200m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendAttributes(&mount, "generic", "hello world", "60m", "200m"),
					testAccCheckVaultSecretBackendConfigAttributes(path, "60m", "200m"),
				),
			},
			// Change just one TTL
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(path, "hello world", "60m", "300m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendAttributes(&mount, "generic", "hello world", "60m", "300m"),
					testAccCheckVaultSecretBackendConfigAttributes(path, "60m", "300m"),
				),
			},
		},
	})
}

func TestAccVaultSecretBackend_implicitParams(t *testing.T) {
	var mount api.MountOutput
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretBackendConfigMinimal(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendAttributes(&mount, "generic", "Managed by Terraform", "0", "0"),
					testAccCheckVaultSecretBackendConfigAttributes("generic", "720h", "720h"),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "path", "generic"),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "default_lease_ttl", "720h0m0s"),
					resource.TestCheckResourceAttr("vault_secret_backend.foo", "max_lease_ttl", "720h0m0s"),
				),
			},
		},
	})
}

func TestAccVaultSecretBackend_updatePath(t *testing.T) {
	var mount api.MountOutput
	pathOne := fmt.Sprintf("path-%s/secret-%s",
		acctest.RandString(5), acctest.RandString(10))
	pathTwo := fmt.Sprintf("%s-updated", pathOne)
	secretPath := "super/secret"

	// Prove path updates do a remount and preserves secrets by writing a secret
	// before update and ensuring it remains intact after path is updated.
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(pathOne, "hello world", "30m", "100m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendConfigAttributes(pathOne, "30m", "100m"),
					testAccCheckVaultWriteSecret(
						strings.Join([]string{pathOne, secretPath}, "/"), "hithere"),
					testAccCheckVaultAssertSecret(
						strings.Join([]string{pathOne, secretPath}, "/"), "hithere"),
				),
			},
			resource.TestStep{
				Config: testAccVaultSecretBackendConfig(pathTwo, "hello world", "30m", "100m"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretBackendExists("vault_secret_backend.foo", &mount),
					testAccCheckVaultSecretBackendConfigAttributes(pathTwo, "30m", "100m"),
					testAccCheckVaultAssertSecret(
						strings.Join([]string{pathTwo, secretPath}, "/"), "hithere"),
				),
			},
		},
	})
}

func testAccCheckVaultWriteSecret(path, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		_, err := client.Logical().Write(path, map[string]interface{}{"value": value})
		return err
	}
}

func testAccCheckVaultAssertSecret(path, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		secret, err := client.Logical().Read(path)
		if err != nil {
			return err
		}
		if secret == nil || secret.Data == nil {
			return fmt.Errorf("No secret found! Expected secret with: %q", value)
		}
		val, ok := secret.Data["value"]
		if !ok {
			return fmt.Errorf("Value not found! Expected: %q", value)
		}
		if val.(string) != value {
			return fmt.Errorf("Expected: %q, got: %q", value, val.(string))
		}
		return nil
	}
}

func testAccCheckVaultSecretBackendExists(key string, mount *api.MountOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[key]
		client := testAccProvider.Meta().(*api.Client)

		mounts, err := client.Sys().ListMounts()
		if err != nil {
			return fmt.Errorf("Error listing mounts: %s", err)
		}

		// Mounts from the API include an extra trailing slash
		m, ok := mounts[fmt.Sprintf("%s/", rs.Primary.ID)]
		if !ok {
			return fmt.Errorf("Mount not found: %s", rs.Primary.ID)
		}

		*mount = *m
		return nil
	}
}

func testAccCheckVaultSecretBackendAttributes(
	mount *api.MountOutput,
	expectedType, expectedDescrip, expectedDefaultTTL, expectedMaxTTL string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mount.Type != expectedType {
			return fmt.Errorf("Expected mount type %q, got %q",
				expectedType, mount.Type)
		}

		if mount.Description != expectedDescrip {
			return fmt.Errorf("Expected mount description %q, got %q",
				expectedDescrip, mount.Description)
		}

		{
			expected, err := time.ParseDuration(expectedDefaultTTL)
			if err != nil {
				return err
			}
			if mount.Config.DefaultLeaseTTL != int(expected.Seconds()) {
				return fmt.Errorf("Expected default lease TTL: %d, got %d",
					int(expected.Seconds()), mount.Config.DefaultLeaseTTL)
			}
		}

		{
			expected, err := time.ParseDuration(expectedMaxTTL)
			if err != nil {
				return err
			}
			if mount.Config.MaxLeaseTTL != int(expected.Seconds()) {
				return fmt.Errorf("Expected max lease TTL: %d, got %d",
					int(expected.Seconds()), mount.Config.MaxLeaseTTL)
			}
			return nil
		}
	}
}

func testAccCheckVaultSecretBackendConfigAttributes(
	path string,
	expectedDefaultTTL, expectedMaxTTL string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		mountConfig, err := client.Sys().MountConfig(path)
		if err != nil {
			return err
		}

		{
			expected, err := time.ParseDuration(expectedDefaultTTL)
			if err != nil {
				return err
			}
			if mountConfig.DefaultLeaseTTL != int(expected.Seconds()) {
				return fmt.Errorf("Expected default lease TTL: %d, got %d",
					int(expected.Seconds()), mountConfig.DefaultLeaseTTL)
			}
		}

		{
			expected, err := time.ParseDuration(expectedMaxTTL)
			if err != nil {
				return err
			}
			if mountConfig.MaxLeaseTTL != int(expected.Seconds()) {
				return fmt.Errorf("Expected max lease TTL: %d, got %d",
					int(expected.Seconds()), mountConfig.MaxLeaseTTL)
			}
			return nil
		}
	}
}

func testAccCheckVaultSecretBackendDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*api.Client)

	existingMounts, err := client.Sys().ListMounts()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_secret_backend" {
			continue
		}
		for mountPoint := range existingMounts {
			if mountPoint == rs.Primary.ID {
				return fmt.Errorf("Mount still exists: %s", mountPoint)
			}
		}
	}

	return nil
}

func testAccVaultSecretBackendDisappear(path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		return client.Sys().Unmount(path)
	}
}

func testAccVaultSecretBackendConfig(
	path, descrip, defaultLeaseTTL, maxLeaseTTL string) string {
	return fmt.Sprintf(`
resource "vault_secret_backend" "foo" {
	type              = "generic"
	path              = "%s"
	description       = "%s"
	default_lease_ttl = "%s"
	max_lease_ttl     = "%s"
}
`, path, descrip, defaultLeaseTTL, maxLeaseTTL)
}

func testAccVaultSecretBackendConfigMinimal() string {
	return `
resource "vault_secret_backend" "foo" {
	type = "generic"
}
`
}
