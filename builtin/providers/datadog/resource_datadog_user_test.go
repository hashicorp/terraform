package datadog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"gopkg.in/zorkian/go-datadog-api.v2"
)

func TestAccDatadogUser_Updated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogUserConfigRequired,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogUserExists("datadog_user.foo"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "email", "test@example.com"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "handle", "test@example.com"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "name", "Test User"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "verified", "false"),
				),
			},
			resource.TestStep{
				Config: testAccCheckDatadogUserConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogUserExists("datadog_user.foo"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "disabled", "true"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "email", "updated@example.com"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "handle", "test@example.com"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "is_admin", "true"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "name", "Updated User"),
					resource.TestCheckResourceAttr(
						"datadog_user.foo", "verified", "false"),
				),
			},
		},
	})
}

func testAccCheckDatadogUserDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)

	if err := datadogUserDestroyHelper(s, client); err != nil {
		return err
	}
	return nil
}

func testAccCheckDatadogUserExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*datadog.Client)
		if err := datadogUserExistsHelper(s, client); err != nil {
			return err
		}
		return nil
	}
}

const testAccCheckDatadogUserConfigRequired = `
resource "datadog_user" "foo" {
  email  = "test@example.com"
  handle = "test@example.com"
  name   = "Test User"
}
`

const testAccCheckDatadogUserConfigUpdated = `
resource "datadog_user" "foo" {
  disabled = true
  email    = "updated@example.com"
  handle   = "test@example.com"
  is_admin = true
  name     = "Updated User"
}
`

func datadogUserDestroyHelper(s *terraform.State, client *datadog.Client) error {
	for _, r := range s.RootModule().Resources {
		id := r.Primary.ID
		u, err := client.GetUser(id)

		if err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				continue
			}
			return fmt.Errorf("Received an error retrieving user %s", err)
		}

		// Datadog only disables user on DELETE
		if u.GetDisabled() {
			continue
		}
		return fmt.Errorf("User still exists")
	}
	return nil
}

func datadogUserExistsHelper(s *terraform.State, client *datadog.Client) error {
	for _, r := range s.RootModule().Resources {
		id := r.Primary.ID
		if _, err := client.GetUser(id); err != nil {
			return fmt.Errorf("Received an error retrieving user %s", err)
		}
	}
	return nil
}
