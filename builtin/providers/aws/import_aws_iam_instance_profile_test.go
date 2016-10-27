package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSIAMInstanceProfile_importBasic(t *testing.T) {
	resourceName := "aws_iam_instance_profile.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSInstanceProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsIamInstanceProfileConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
