package icinga2

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func TestAccCreateCheckCommand(t *testing.T) {

	var testAccCreateBasicCheckCommand = fmt.Sprintf(`
		resource "icinga2_checkcommand" "basic" {
		name      = "terraform-test-checkcommand-1"
		templates = [ "plugin-check-command" ]
		command = "/usr/local/bin/check_command"
    arguments = {
		  "-I" = "$IARG$"
			"-J" = "$JARG$" }
	}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateBasicCheckCommand,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCheckCommandExists("icinga2_checkcommand.basic"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "name", "terraform-test-checkcommand-1"),
					//testAccCheckResourceState("icinga2_checkcommand.basic", "command.#", "1"),
					//testAccCheckResourceState("icinga2_checkcommand.basic", "command.0", "/usr/local/bin/check_command"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "command", "/usr/local/bin/check_command"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.%", "2"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.-I", "$IARG$"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.-J", "$JARG$"),
				),
			},
		},
	})
}

func testAccCheckCheckCommandExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("CheckCommand resource not found: %s", rn)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("CheckCommand resource id not set")
		}

		client := testAccProvider.Meta().(*iapi.Server)
		_, err := client.GetCheckCommand(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error getting getting CheckCommand: %s", err)
		}

		return nil
	}
}
