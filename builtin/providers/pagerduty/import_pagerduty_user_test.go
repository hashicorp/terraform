package pagerduty

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyUser_import(t *testing.T) {
	resourceName := "pagerduty_user.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyUserConfig,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
