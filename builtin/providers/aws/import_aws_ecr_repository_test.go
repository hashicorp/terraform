package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEcrRepository_importBasic(t *testing.T) {
	resourceName := "aws_ecr_repository.default"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcrRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcrRepository,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
