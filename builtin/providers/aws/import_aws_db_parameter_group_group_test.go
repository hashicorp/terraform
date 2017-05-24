package aws

import (
	"fmt"
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSDBParameterGroup_importBasic(t *testing.T) {
	resourceName := "aws_db_parameter_group.bar"
	groupName := fmt.Sprintf("parameter-group-test-terraform-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBParameterGroupConfig(groupName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
