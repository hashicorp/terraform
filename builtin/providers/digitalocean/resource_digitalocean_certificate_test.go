package digitalocean

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanCertificate_Basic(t *testing.T) {
	var cert godo.Certificate
	rInt := acctest.RandInt()
	leafCertMaterial, privateKeyMaterial, err := acctest.RandTLSCert("Acme Co")
	if err != nil {
		t.Fatalf("Cannot generate test TLS certificate: %s", err)
	}
	rootCertMaterial, _, err := acctest.RandTLSCert("Acme Go")
	if err != nil {
		t.Fatalf("Cannot generate test TLS certificate: %s", err)
	}
	certChainMaterial := fmt.Sprintf("%s\n%s", strings.TrimSpace(rootCertMaterial), leafCertMaterial)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanCertificateConfig_basic(rInt, privateKeyMaterial, leafCertMaterial, certChainMaterial),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanCertificateExists("digitalocean_certificate.foobar", &cert),
					resource.TestCheckResourceAttr(
						"digitalocean_certificate.foobar", "name", fmt.Sprintf("certificate-%d", rInt)),
					resource.TestCheckResourceAttr(
						"digitalocean_certificate.foobar", "private_key", fmt.Sprintf("%s\n", privateKeyMaterial)),
					resource.TestCheckResourceAttr(
						"digitalocean_certificate.foobar", "leaf_certificate", fmt.Sprintf("%s\n", leafCertMaterial)),
					resource.TestCheckResourceAttr(
						"digitalocean_certificate.foobar", "certificate_chain", fmt.Sprintf("%s\n", certChainMaterial)),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanCertificateDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_certificate" {
			continue
		}

		_, _, err := client.Certificates.Get(context.Background(), rs.Primary.ID)

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for certificate (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckDigitalOceanCertificateExists(n string, cert *godo.Certificate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Certificate ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		c, _, err := client.Certificates.Get(context.Background(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if c.ID != rs.Primary.ID {
			return fmt.Errorf("Certificate not found")
		}

		*cert = *c

		return nil
	}
}

func testAccCheckDigitalOceanCertificateConfig_basic(rInt int, privateKeyMaterial, leafCert, certChain string) string {
	return fmt.Sprintf(`
resource "digitalocean_certificate" "foobar" {
  name = "certificate-%d"
  private_key = <<EOF
%s
EOF
  leaf_certificate = <<EOF
%s
EOF
  certificate_chain = <<EOF
%s
EOF
}`, rInt, privateKeyMaterial, leafCert, certChain)
}
