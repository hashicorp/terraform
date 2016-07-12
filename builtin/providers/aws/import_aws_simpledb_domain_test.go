package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSimpleDBDomain_importBasic(t *testing.T) {
	resourceName := "aws_simpledb_domain.test_domain"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSimpleDBDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSimpleDBDomainConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
