package opsgenie

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/user"
)

func TestAccOpsGenieUser_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMContainerRegistry_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieUserExists("opsgenie_user.test"),
				),
			},
		},
	})
}

func testCheckOpsGenieUserDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.OpsGenieClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opsgenie_user" {
			continue
		}

		userClient, err := conn.User()
		if err != nil {
			return err
		}

		req := user.GetUserRequest{
			Id: rs.Primary.Attributes["id"],
		}

		result, err := userClient.Get(req)
		if result != nil {
			return fmt.Errorf("User still exists:\n%#v", result)
		}
	}

	return nil
}

func testCheckOpsGenieUserExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.Attributes["id"]
		username := rs.Primary.Attributes["username"]

		conn := testAccProvider.Meta().(*client.OpsGenieClient)
		userClient, err := conn.User()
		if err != nil {
			return err
		}

		req := user.GetUserRequest{
			Id: rs.Primary.Attributes["id"],
		}

		result, err := userClient.Get(req)
		if result == nil {
			return fmt.Errorf("Bad: User %q (username: %q) does not exist", id, username)
		}

		return nil
	}
}

var testAccAzureRMContainerRegistry_basic = `
resource "opsgenie_user" "test" {
  username  = "acctest-%d@example.tld"
  full_name = "Acceptance Test User"
  role      = "User"
}
`
