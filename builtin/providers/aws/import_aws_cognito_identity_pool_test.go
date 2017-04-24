package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCognitoIdentityPool_importBasic(t *testing.T) {
	resourceName := "aws_cognito_identity_pool.main"
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(rName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
