package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDBClusterParameterGroup_importBasic(t *testing.T) {
	resourceName := "aws_rds_cluster_parameter_group.bar"

	parameterGroupName := fmt.Sprintf("cluster-parameter-group-test-terraform-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBClusterParameterGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBClusterParameterGroupConfig(parameterGroupName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
