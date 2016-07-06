package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCloudFrontOriginAccessIdentity_importBasic(t *testing.T) {
	resourceName := "aws_cloudfront_origin_access_identity.origin_access_identity"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontOriginAccessIdentityDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFrontOriginAccessIdentityConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
