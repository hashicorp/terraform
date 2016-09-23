package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyUser_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPagerDutyUserConfigImported(importUserID),
			},
			resource.TestStep{
				ResourceName:      "pagerduty_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPagerDutyUserConfigImported(id string) string {
	return fmt.Sprintf(`
		resource "pagerduty_user" "foo" {
		  name = "foo"
			email = "foo@bar.com"
   	}
	`)
}
