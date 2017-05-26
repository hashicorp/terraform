package icinga2

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func TestAccCreateCheckcommand(t *testing.T) {

	var testAccCreateCheckcommand = fmt.Sprintf(`
		resource "icinga2_checkcommand" "checkcommand" {
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
				Config: testAccCreateCheckcommand,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCheckcommandExists("icinga2_checkcommand.checkcommand"),
					testAccCheckResourceState("icinga2_checkcommand.checkcommand", "name", "terraform-test-checkcommand-1"),
					testAccCheckResourceState("icinga2_checkcommand.checkcommand", "command", "/usr/local/bin/check_command"),
					testAccCheckResourceState("icinga2_checkcommand.checkcommand", "arguments.%", "2"),
					testAccCheckResourceState("icinga2_checkcommand.checkcommand", "arguments.-I", "$IARG$"),
					testAccCheckResourceState("icinga2_checkcommand.checkcommand", "arguments.-J", "$JARG$"),
				),
			},
		},
	})
}

func testAccCheckCheckcommandExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Checkcommand resource not found: %s", rn)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Checkcommand resource id not set")
		}

		client := testAccProvider.Meta().(*iapi.Server)
		_, err := client.GetCheckcommand(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error getting getting Checkcommand: %s", err)
		}

		return nil
	}
}
