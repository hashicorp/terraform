package pagerduty

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutyOnCall_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPagerDutyOnCallsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccPagerDutyOnCalls("data.pagerduty_on_call.foo"),
				),
			},
		},
	})
}

func testAccPagerDutyOnCalls(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		var size int
		var err error

		if size, err = strconv.Atoi(a["oncalls.#"]); err != nil {
			return err
		}

		if size == 0 {
			return fmt.Errorf("Expected at least one on call in the list. Found: %d", size)
		}

		for i := range make([]string, size) {
			escalationLevel := a[fmt.Sprintf("oncalls.%d.escalation_level", i)]
			if escalationLevel == "" {
				return fmt.Errorf("Expected the on call to have an escalation_level set")
			}
		}

		return nil
	}
}

const testAccPagerDutyOnCallsConfig = `
data "pagerduty_on_call" "foo" {}
`
