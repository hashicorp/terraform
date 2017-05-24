package rancher

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherRegistrationToken_basic(t *testing.T) {
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

func TestAccRancherRegistrationToken_disappears(t *testing.T) {
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
					testAccRancherRegistrationTokenDisappears(&registrationToken),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRancherRegistrationTokenDisappears(token *rancherClient.RegistrationToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(token.AccountId)
		if err != nil {
			return err
		}

		if _, e := client.RegistrationToken.ActionDeactivate(token); e != nil {
			return fmt.Errorf("Error deactivating RegistrationToken: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"active", "inactive", "deactivating"},
			Target:     []string{"inactive"},
			Refresh:    RegistrationTokenStateRefreshFunc(client, token.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, waitErr := stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for registration token (%s) to be deactivated: %s", token.Id, waitErr)
		}

		// Update resource to reflect its state
		token, err = client.RegistrationToken.ById(token.Id)
		if err != nil {
			return fmt.Errorf("Failed to refresh state of deactivated registration token (%s): %s", token.Id, err)
		}

		// Step 2: Remove
		if _, err := client.RegistrationToken.ActionRemove(token); err != nil {
			return fmt.Errorf("Error removing RegistrationToken: %s", err)
		}

		stateConf = &resource.StateChangeConf{
			Pending:    []string{"inactive", "removed", "removing"},
			Target:     []string{"removed"},
			Refresh:    RegistrationTokenStateRefreshFunc(client, token.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, waitErr = stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for registration token (%s) to be removed: %s", token.Id, waitErr)
		}

		return nil
	}
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

		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(rs.Primary.Attributes["environment_id"])
		if err != nil {
			return err
		}

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

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_registration_token" {
			continue
		}
		client, err := testAccProvider.Meta().(*Config).GlobalClient()
		if err != nil {
			return err
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
