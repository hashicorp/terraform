package rancher

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherRegistrationToken(t *testing.T) {
	var registrationToken rancherClient.RegistrationToken

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherRegistrationTokenDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherRegistrationTokenConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistrationTokenExists("rancher_registration_token.foo", &registrationToken),
					resource.TestCheckResourceAttr(
						"rancher_registration_token.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"rancher_registration_token.foo", "description", "Terraform acc test group"),
					resource.TestCheckResourceAttrSet("rancher_registration_token.foo", "command"),
					resource.TestCheckResourceAttrSet("rancher_registration_token.foo", "registration_url"),
					resource.TestCheckResourceAttrSet("rancher_registration_token.foo", "token"),
				),
			},
			resource.TestStep{
				Config: testAccRancherRegistrationTokenUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherRegistrationTokenExists("rancher_registration_token.foo", &registrationToken),
					resource.TestCheckResourceAttr(
						"rancher_registration_token.foo", "name", "foo-u"),
					resource.TestCheckResourceAttr(
						"rancher_registration_token.foo", "description", "Terraform acc test group-u"),
					resource.TestCheckResourceAttrSet("rancher_registration_token.foo", "command"),
					resource.TestCheckResourceAttrSet("rancher_registration_token.foo", "registration_url"),
					resource.TestCheckResourceAttrSet("rancher_registration_token.foo", "token"),
				),
			},
		},
	})
}

func testAccCheckRancherRegistrationTokenExists(n string, regT *rancherClient.RegistrationToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*Config)

		foundRegT, err := client.RegistrationToken.ById(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundRegT.Resource.Id != rs.Primary.ID {
			return fmt.Errorf("RegistrationToken not found")
		}

		*regT = *foundRegT

		return nil
	}
}

func testAccCheckRancherRegistrationTokenDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_registration_token" {
			continue
		}
		regT, err := client.RegistrationToken.ById(rs.Primary.ID)

		if err == nil {
			if regT != nil &&
				regT.Resource.Id == rs.Primary.ID &&
				regT.State != "removed" {
				return fmt.Errorf("RegistrationToken still exists")
			}
		}

		return nil
	}
	return nil
}

const testAccRancherRegistrationTokenConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_registration_token" "foo" {
	name = "foo"
	description = "Terraform acc test group"
	environment_id = "${rancher_environment.foo.id}"
}
`

const testAccRancherRegistrationTokenUpdateConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_registration_token" "foo" {
	name = "foo-u"
	description = "Terraform acc test group-u"
	environment_id = "${rancher_environment.foo.id}"
}
`
