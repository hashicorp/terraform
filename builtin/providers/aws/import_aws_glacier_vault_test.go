package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSGlacierVault_importBasic(t *testing.T) {
	resourceName := "aws_glacier_vault.full"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlacierVaultDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGlacierVault_full,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
