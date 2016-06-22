package aws

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcrRepository_importBasic(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 1: repository
		if len(s) != 1 {
			return fmt.Errorf("bad states: %#v", s)
		}

		return nil
	}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcrRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcrRepository,
			},

			resource.TestStep{
				ResourceName:     "aws_ecr_repository.default",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}
