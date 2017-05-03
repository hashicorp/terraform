package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEmrSecurityConfiguration_importBasic(t *testing.T) {
	resourceName := "aws_emr_security_configuration.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEmrSecurityConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEmrSecurityConfigurationConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
