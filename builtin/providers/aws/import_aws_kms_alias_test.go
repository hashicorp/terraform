package aws

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSKmsAlias_importBasic(t *testing.T) {
	resourceName := "aws_kms_alias.single"
	rInt := acctest.RandInt()
	kmsAliasTimestamp := time.Now().Format(time.RFC1123)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsSingleAlias(rInt, kmsAliasTimestamp),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
