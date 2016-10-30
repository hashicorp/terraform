package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAWSElasticBeanstalkEnvironment_importBasic(t *testing.T) {
	resourceName := "aws_elastic_beanstalk_application.tftest"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnvConfig,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
