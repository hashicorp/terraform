package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSFlowLog_importBasic(t *testing.T) {
	resourceName := "aws_flow_log.test_flow_log"

	fln := fmt.Sprintf("tf-test-fl-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFlowLogConfig_basic(fln),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
