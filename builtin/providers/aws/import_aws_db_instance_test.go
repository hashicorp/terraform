package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDBInstance_importBasic(t *testing.T) {
	resourceName := "aws_db_instance.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBInstanceConfig,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_final_snapshot",
					"final_snapshot_identifier",
				},
			},
		},
	})
}
