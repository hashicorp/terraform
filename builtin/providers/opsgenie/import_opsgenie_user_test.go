package opsgenie

import (
	"testing"

	"fmt"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccOpsGenieUser_importBasic(t *testing.T) {
	resourceName := "opsgenie_user.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieUser_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpsGenieUser_importComplete(t *testing.T) {
	resourceName := "opsgenie_user.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieUser_complete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
