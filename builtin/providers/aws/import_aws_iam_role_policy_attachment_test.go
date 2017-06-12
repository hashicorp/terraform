package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsIamRolePolicyAttachmentImport(t *testing.T) {
	resourceName := "aws_iam_role_policy_attachment.test-attach"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRolePolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRolePolicyAttachConfig,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "test-role",
				ImportStateVerify: true,
			},
		},
	})
}
