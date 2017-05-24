package aws

import (
	"fmt"
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSElasticacheParameterGroup_importBasic(t *testing.T) {
	resourceName := "aws_elasticache_parameter_group.bar"
	rName := fmt.Sprintf("parameter-group-test-terraform-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheParameterGroupConfig(rName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
