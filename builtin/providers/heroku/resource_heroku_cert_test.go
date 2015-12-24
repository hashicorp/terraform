package heroku

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuCert_Basic(t *testing.T) {
	var endpoint heroku.SSLEndpoint
	wd, _ := os.Getwd()
	certificateChainFile := wd + "/test-fixtures/terraform.cert"
	certificateChainBytes, _ := ioutil.ReadFile(certificateChainFile)
	certificateChain := string(certificateChainBytes)
	testAccCheckHerokuCertConfig_basic := `
    resource "heroku_app" "foobar" {
        name = "terraform-test-cert-app"
        region = "eu"
    }

    resource "heroku_addon" "ssl" {
        app = "${heroku_app.foobar.name}"
        plan = "ssl:endpoint"
    }

    resource "heroku_cert" "ssl_certificate" {
        app = "${heroku_app.foobar.name}"
        depends_on = ["heroku_addon.ssl"]
        certificate_chain="${file("` + certificateChainFile + `")}"
        private_key="${file("` + wd + `/test-fixtures/terraform.key")}"
    }
    `

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuCertDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckHerokuCertConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
					resource.TestCheckResourceAttr(
						"heroku_cert.ssl_certificate", "cname", "terraform-test-cert-app.herokuapp.com"),
				),
			},
		},
	})
}

func testAccCheckHerokuCertDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_cert" {
			continue
		}

		_, err := client.SSLEndpointInfo(rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Cerfificate still exists")
		}
	}

	return nil
}

func testAccCheckHerokuCertificateChain(endpoint *heroku.SSLEndpoint, chain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if endpoint.CertificateChain != chain {
			return fmt.Errorf("Bad certificate chain: %s", endpoint.CertificateChain)
		}

		return nil
	}
}

func testAccCheckHerokuCertExists(n string, endpoint *heroku.SSLEndpoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSL endpoint ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundEndpoint, err := client.SSLEndpointInfo(rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundEndpoint.ID != rs.Primary.ID {
			return fmt.Errorf("SSL endpoint not found")
		}

		*endpoint = *foundEndpoint

		return nil
	}
}
