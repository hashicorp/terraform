package vault

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccVaultSecret_basic(t *testing.T) {
	var secret api.Secret
	backendPath := fmt.Sprintf("secret-%s", acctest.RandString(6))
	secretPath := fmt.Sprintf("foo-%s", acctest.RandString(4))
	data := map[string]string{"hush": "secret"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretConfigGeneric(
					backendPath, secretPath, "10m", data),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretExists("vault_secret.foo", &secret),
					testAccCheckVaultSecretAttributes(&secret, "10m", data),
				),
			},
		},
	})
}

func TestAccVaultSecret_cubbyhole(t *testing.T) {
	var secret api.Secret
	secretPath := fmt.Sprintf("foo-%s", acctest.RandString(4))
	data := map[string]string{"hush": "secret"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretConfigCubbyhole(secretPath, data),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretExists("vault_secret.foo", &secret),
					testAccCheckVaultSecretAttributes(&secret, "", data),
				),
			},
		},
	})
}

func TestAccVaultSecret_disappears(t *testing.T) {
	var secret api.Secret
	backendPath := fmt.Sprintf("secret-%s", acctest.RandString(6))
	secretPath := fmt.Sprintf("foo-%s", acctest.RandString(4))
	data := map[string]string{"hush": "secret"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretConfigGeneric(
					backendPath, secretPath, "10m", data),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretExists("vault_secret.foo", &secret),
					testAccVaultSecretDisappear(
						strings.Join([]string{backendPath, secretPath}, "/")),
				),
				ExpectNonEmptyPlan: true,
			},
			// Empty config should yield empty plan, since token is gone
			resource.TestStep{
				Config: "",
			},
		},
	})
}

func TestAccVaultSecret_dataDrift(t *testing.T) {
	var secret api.Secret
	backendPath := fmt.Sprintf("secret-%s", acctest.RandString(6))
	secretPath := fmt.Sprintf("foo-%s", acctest.RandString(4))
	data := map[string]string{"hush": "secret"}
	dataDrift := map[string]interface{}{"hush": "secret", "manually": "changed"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretConfigGeneric(
					backendPath, secretPath, "10m", data),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretExists("vault_secret.foo", &secret),
					testAccVaultSecretDrift(
						strings.Join([]string{backendPath, secretPath}, "/"), dataDrift),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccVaultSecret_ttlDrift(t *testing.T) {
	var secret api.Secret
	backendPath := fmt.Sprintf("secret-%s", acctest.RandString(6))
	secretPath := fmt.Sprintf("foo-%s", acctest.RandString(4))
	data := map[string]string{"hush": "secret"}
	ttlDrift := map[string]interface{}{"hush": "secret", "ttl": "11m"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultSecretDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultSecretConfigGeneric(
					backendPath, secretPath, "10m", data),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultSecretExists("vault_secret.foo", &secret),
					testAccVaultSecretDrift(
						strings.Join([]string{backendPath, secretPath}, "/"), ttlDrift),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckVaultSecretExists(key string, secret *api.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[key]
		client := testAccProvider.Meta().(*api.Client)

		t, err := client.Logical().Read(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error looking up secret: %s", err)
		}

		*secret = *t
		return nil
	}
}

func testAccCheckVaultSecretAttributes(
	secret *api.Secret,
	expectedTTL string,
	expectedData map[string]string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expectedTTL != "" {
			expectedDuration, err := time.ParseDuration(expectedTTL)
			if err != nil {
				return err
			}
			gotDuration := time.Duration(secret.LeaseDuration) * time.Second
			if gotDuration != expectedDuration {
				return fmt.Errorf("Expected TTL %s, got %d",
					expectedDuration, gotDuration)
			}
		}

		gotData := make(map[string]string)
		for k, v := range secret.Data {
			if k != "ttl" {
				gotData[k] = v.(string)
			}
		}
		if !reflect.DeepEqual(gotData, expectedData) {
			return fmt.Errorf("Expected data %v, got %v", expectedData, gotData)
		}

		return nil
	}
}

func testAccCheckVaultSecretDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*api.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_secret" {
			continue
		}

		secret, err := client.Logical().Read(rs.Primary.ID)
		if err != nil {
			return err
		}

		if secret != nil {
			return fmt.Errorf("Secret still exists: %#v", secret)
		}
	}

	return nil
}

func testAccVaultSecretDisappear(path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		_, err := client.Logical().Delete(path)
		return err
	}
}

func testAccVaultSecretDrift(
	path string, data map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		_, err := client.Logical().Write(path, data)
		return err
	}
}

func testAccVaultSecretConfigCubbyhole(
	secretPath string,
	data map[string]string) string {
	var d bytes.Buffer
	for k, v := range data {
		d.WriteString(fmt.Sprintf("    %s = %q\n", k, v))
	}
	return fmt.Sprintf(`
resource "vault_secret" "foo" {
  path = "cubbyhole/%s"
  data {
    %s
  }
}
`, secretPath, d.String())
}

func testAccVaultSecretConfigGeneric(
	backendPath string,
	secretPath string,
	ttl string,
	data map[string]string) string {
	var d bytes.Buffer
	for k, v := range data {
		d.WriteString(fmt.Sprintf("    %s = %q\n", k, v))
	}
	if ttl != "" {
		ttl = fmt.Sprintf("ttl = %q", ttl)
	}
	return fmt.Sprintf(`
resource "vault_secret_backend" "foo" {
  type = "generic"
  path = "%s"
}
resource "vault_secret" "foo" {
  path = "${vault_secret_backend.foo.path}/%s"
  %s
  data {
    %s
  }
}
`, backendPath, secretPath, ttl, d.String())
}

func testAccVaultSecretConfigMinimal() string {
	return `
resource "vault_secret" "foo" {
}
`
}
