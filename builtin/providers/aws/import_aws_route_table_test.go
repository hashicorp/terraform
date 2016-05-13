package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRouteTable_importBasic(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 2: group, 1 rules
		if len(s) != 2 {
			return fmt.Errorf("bad states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRouteTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRouteTableConfig,
			},

			resource.TestStep{
				ResourceName:     "aws_route_table.foo",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}
