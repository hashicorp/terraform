package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSRoute53HealthCheck_importBasic(t *testing.T) {
	resourceName := "aws_route53_health_check.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53HealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53HealthCheckConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
