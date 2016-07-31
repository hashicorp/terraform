package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSKeyPair_importBasic(t *testing.T) {
	resourceName := "aws_key_pair.a_key_pair"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKeyPairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKeyPairConfig,
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"public_key"},
			},
		},
	})
}
