package vault

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccVaultAuditBackend_basic(t *testing.T) {
	var audit api.Audit
	auditPath := fmt.Sprintf("path-%s/audit-%s",
		acctest.RandString(5), acctest.RandString(10))
	dir, err := ioutil.TempDir("", "tfacctests")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)
	filePath := path.Join(dir, fmt.Sprintf("%s.log", acctest.RandString(8)))
	options := map[string]string{"path": filePath}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultAuditBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultAuditBackendConfig(
					"file", auditPath, "hello audit", options),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultAuditBackendExists("vault_audit_backend.foo", &audit),
					testAccCheckVaultAuditBackendAttributes(
						&audit, "file", "hello audit", options),
				),
			},
		},
	})
}

func TestAccVaultAuditBackend_disappears(t *testing.T) {
	var audit api.Audit
	auditPath := fmt.Sprintf("path-%s/audit-%s",
		acctest.RandString(5), acctest.RandString(10))
	dir, err := ioutil.TempDir("", "tfacctests")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)
	filePath := path.Join(dir, fmt.Sprintf("%s.log", acctest.RandString(8)))
	options := map[string]string{"path": filePath}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultAuditBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultAuditBackendConfig(
					"file", auditPath, "hello audit", options),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultAuditBackendExists("vault_audit_backend.foo", &audit),
					testAccVaultAuditBackendDisappear(auditPath),
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

func TestAccVaultAuditBackend_implicitParams(t *testing.T) {
	var audit api.Audit
	dir, err := ioutil.TempDir("", "tfacctests")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)
	filePath := path.Join(dir, fmt.Sprintf("%s.log", acctest.RandString(8)))
	options := map[string]string{"path": filePath}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultAuditBackendDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultAuditBackendConfigMinimal("file", options),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultAuditBackendExists("vault_audit_backend.foo", &audit),
					testAccCheckVaultAuditBackendAttributes(
						&audit, "file", "Managed by Terraform", options),
					resource.TestCheckResourceAttr(
						"vault_audit_backend.foo", "path", "file"),
				),
			},
		},
	})
}

func testAccCheckVaultAuditBackendExists(
	key string, audit *api.Audit) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[key]
		client := testAccProvider.Meta().(*api.Client)

		audits, err := client.Sys().ListAudit()
		if err != nil {
			return fmt.Errorf("Error listing mounts: %s", err)
		}

		// Paths from the API include an extra trailing slash
		a, ok := audits[fmt.Sprintf("%s/", rs.Primary.ID)]
		if !ok {
			return fmt.Errorf("audit backend not found: %s", rs.Primary.ID)
		}

		*audit = *a
		return nil
	}
}

func testAccCheckVaultAuditBackendAttributes(
	audit *api.Audit,
	expectedType, expectedDescrip string,
	expectedOptions map[string]string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if audit.Type != expectedType {
			return fmt.Errorf("Expected audit type %q, got %q",
				expectedType, audit.Type)
		}
		if audit.Description != expectedDescrip {
			return fmt.Errorf("Expected audit description %q, got %q",
				expectedDescrip, audit.Description)
		}
		if !reflect.DeepEqual(audit.Options, expectedOptions) {
			return fmt.Errorf("Expected audit options %v, got %v",
				expectedOptions, audit.Options)
		}
		return nil
	}
}

func testAccCheckVaultAuditBackendDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*api.Client)

	existingAuditBackends, err := client.Sys().ListAudit()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_audit_backend" {
			continue
		}
		for mountPoint := range existingAuditBackends {
			if mountPoint == rs.Primary.ID {
				return fmt.Errorf("AuditBackend still exists: %s", mountPoint)
			}
		}
	}

	return nil
}

func testAccVaultAuditBackendDisappear(path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		return client.Sys().DisableAudit(path)
	}
}

func testAccVaultAuditBackendConfig(
	auditType, path, descrip string,
	options map[string]string) string {
	var opts bytes.Buffer
	for k, v := range options {
		opts.WriteString(fmt.Sprintf("    %s = %q\n", k, v))
	}
	return fmt.Sprintf(`
resource "vault_audit_backend" "foo" {
  type              = "%s"
  path              = "%s"
  description       = "%s"
  options {
%s
  }
}
`, auditType, path, descrip, opts.String())
}

func testAccVaultAuditBackendConfigMinimal(
	auditType string, options map[string]string) string {
	var opts bytes.Buffer
	for k, v := range options {
		opts.WriteString(fmt.Sprintf("    %s = %q\n", k, v))
	}
	return fmt.Sprintf(`
resource "vault_audit_backend" "foo" {
  type = "%s"
  options {
%s
  }
}
`, auditType, opts.String())
}
