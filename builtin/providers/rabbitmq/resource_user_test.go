package rabbitmq

import (
	"fmt"
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUser(t *testing.T) {
	var user string
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccUserCheckDestroy(user),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccUserConfig_basic,
				Check: testAccUserCheck(
					"rabbitmq_user.test", &user,
				),
			},
			resource.TestStep{
				Config: testAccUserConfig_update,
				Check: testAccUserCheck(
					"rabbitmq_user.test", &user,
				),
			},
		},
	})
}

func testAccUserCheck(rn string, name *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("user id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		users, err := rmqc.ListUsers()
		if err != nil {
			return fmt.Errorf("Error retrieving users: %s", err)
		}

		for _, user := range users {
			if user.Name == rs.Primary.ID {
				*name = rs.Primary.ID
				return nil
			}
		}

		return fmt.Errorf("Unable to find user %s", rn)
	}
}

func testAccUserCheckDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		users, err := rmqc.ListUsers()
		if err != nil {
			return fmt.Errorf("Error retrieving users: %s", err)
		}

		for _, user := range users {
			if user.Name == name {
				return fmt.Errorf("user still exists: %s", name)
			}
		}

		return nil
	}
}

const testAccUserConfig_basic = `
resource "rabbitmq_user" "test" {
    name = "mctest"
    password = "foobar"
    tags = ["administrator", "management"]
}`

const testAccUserConfig_update = `
resource "rabbitmq_user" "test" {
    name = "mctest"
    password = "foobarry"
    tags = ["management"]
}`
