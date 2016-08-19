package icinga2

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCreateBasicHostGroup(t *testing.T) {

	var testAccCreateBasicHostGroup = fmt.Sprintf(`
		resource "icinga2_hostgroup" "basic" {
		  name = "terraform-test-hostgroup"
		  display_name = "Terraform Test HostGroup"
	}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateBasicHostGroup,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_hostgroup.basic"),
					testAccCheckResourceState("icinga2_hostgroup.basic", "name", "terraform-test-hostgroup"),
					testAccCheckResourceState("icinga2_hostgroup.basic", "display_name", "Terraform Test HostGroup"),
				),
			},
		},
	})
}
