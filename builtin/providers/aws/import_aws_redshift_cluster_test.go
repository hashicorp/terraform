package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSRedshiftCluster_importBasic(t *testing.T) {
	resourceName := "aws_redshift_cluster.default"
	config := fmt.Sprintf(testAccAWSRedshiftClusterConfig_basic, acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"master_password", "skip_final_snapshot"},
			},
		},
	})
}
