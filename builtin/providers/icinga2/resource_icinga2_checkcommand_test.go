package icinga2

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCreateBasicCheckcommand(t *testing.T) {

	var testAccCreateBasicCheckcommand = fmt.Sprintf(`
		resource "icinga2_checkcommand" "basic" {
		name      = "terraform-test-checkcommand"
		templates = [ "plugin-check-command" ]
		command = [ "/usr/local/bin/check_command"]
    arguments = {
		  "-I" = "$IARG$"
			"-J" = "$JARG$" }
	}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateBasicCheckcommand,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_checkcommand.basic"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "name", "terraform-test-checkcommand"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "command.#", "1"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "command.0", "/usr/local/bin/check_command"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.#", "2"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.-I", "$IARG$"),
					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.-J", "$JARG$"),
				),
			},
		},
	})
}

//func TestAccCheckcommandModifyArguments(t *testing.T) {
//
//	var testAccCheckcommandModifyArguments = fmt.Sprintf(`
//		resource "icinga2_checkcommand" "basic" {
//		name      = "terraform-test-checkcommand"
//		templates = [ "plugin-check-command" ]
//		command = [ "/usr/local/bin/check_command"]
//    arguments = {
//		  "-I" = "$ARGI$"
//			"-J" = "$JARG$" }
//	}`)
//
//	resource.Test(t, resource.TestCase{
//		Providers: testAccProviders,
//		Steps: []resource.TestStep{
//			resource.TestStep{
//				Config: testAccCheckcommandModifyArguments,
//				Check: resource.ComposeTestCheckFunc(
//					VerifyResourceExists(t, "icinga2_checkcommand.basic"),
//					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.#", "2"),
//					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.-I", "$ARGI$"),
//					testAccCheckResourceState("icinga2_checkcommand.basic", "arguments.-J", "$JARG$"),
//				),
//			},
//		},
//	})
//}
