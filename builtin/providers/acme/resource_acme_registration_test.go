package acme

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccACMERegistration_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckReg(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccACMERegistrationConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckACMERegistrationValid("acme_registration.reg"),
				),
			},
		},
	})
}

func testAccCheckACMERegistrationValid(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find ACME registration: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ACME registration ID not set")
		}

		d := testAccCheckACMERegistrationResourceData(rs)

		client, _, err := expandACMEClient(d, d.Get("registration_url").(string))
		if err != nil {
			return fmt.Errorf("Could not build ACME client off reg: %s", err.Error())
		}
		reg, err := client.QueryRegistration()
		if err != nil {
			return fmt.Errorf("Error on reg query: %s", err.Error())
		}

		actual := reg.URI
		expected := rs.Primary.ID

		if actual != expected {
			return fmt.Errorf("Expected ID to be %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccCheckACMERegistrationResourceData returns a *schema.ResourceData that should match a
// acme_registration resource.
func testAccCheckACMERegistrationResourceData(rs *terraform.ResourceState) *schema.ResourceData {
	r := &schema.Resource{
		Schema: registrationSchemaFull(),
	}
	d := r.TestResourceData()

	d.SetId(rs.Primary.ID)
	d.Set("server_url", rs.Primary.Attributes["server_url"])
	d.Set("account_key_pem", rs.Primary.Attributes["account_key_pem"])
	d.Set("email_address", rs.Primary.Attributes["email_address"])
	d.Set("registration_body", rs.Primary.Attributes["registration_body"])
	d.Set("registration_url", rs.Primary.Attributes["registration_url"])
	d.Set("registration_new_authz_url", rs.Primary.Attributes["registration_new_authz_url"])
	d.Set("registration_tos_url", rs.Primary.Attributes["registration_tos_url"])

	return d
}

func testAccPreCheckReg(t *testing.T) {
	if v := os.Getenv("ACME_EMAIL_ADDRESS"); v == "" {
		t.Fatal("ACME_EMAIL_ADDRESS must be set for the registration acceptance test")
	}
}

func testAccACMERegistrationConfig() string {
	return fmt.Sprintf(`
resource "tls_private_key" "private_key" {
    algorithm = "RSA"
}

resource "acme_registration" "reg" {
	server_url = "https://acme-staging.api.letsencrypt.org/directory"
  account_key_pem = "${tls_private_key.private_key.private_key_pem}"
  email_address = "%s"

}
`, os.Getenv("ACME_EMAIL_ADDRESS"))
}
