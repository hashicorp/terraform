package aws

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSPlacementGroup_importBasic(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 1: placement group
		if len(s) != 1 {
			return fmt.Errorf("bad states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPlacementGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSPlacementGroupConfig,
			},

			resource.TestStep{
				ResourceName:     "aws_placement_group.pg",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}
