package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGoogleSqlSslCert_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleSqlSslCertDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleSqlSslCert_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleSqlSslCertExists("google_sql_ssl_cert.cert"),
				),
			},
		},
	})
}

func testAccCheckGoogleSqlSslCertExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		instance := rs.Primary.Attributes["server_ca_cert.0.instance"]
		sha1 := rs.Primary.Attributes["sha1_fingerprint"]

		_, err := config.clientSqlAdmin.SslCerts.Get(config.Project,
			instance, sha1).Do()

		if err != nil {
			return fmt.Errorf("Not found: %s: %s", n, err)
		}

		return nil
	}
}

func testAccGoogleSqlSslCertDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		config := testAccProvider.Meta().(*Config)
		if rs.Type != "google_sql_ssl_cert" {
			continue
		}

		instance := rs.Primary.Attributes["server_ca_cert.0.instance"]
		sha1 := rs.Primary.Attributes["sha1_fingerprint"]

		_, err := config.clientSqlAdmin.SslCerts.Get(config.Project,
			instance, sha1).Do()

		if err == nil {
			return fmt.Errorf("SslCert resource still exists")
		}
	}

	return nil
}

var testGoogleSqlSslCert_basic = fmt.Sprintf(`
resource "google_sql_database_instance" "instance" {
	name = "tf-lw-%d"
	region = "us-central"
	settings {
		tier = "D0"
	}
}

resource "google_sql_ssl_cert" "cert" {
	server_ca_cert {
		common_name = "mycert"
		instance = "${google_sql_database_instance.instance.name}"
	}
}
`, genRandInt())
