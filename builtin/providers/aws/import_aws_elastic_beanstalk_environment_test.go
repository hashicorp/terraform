package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAWSElasticBeanstalkEnvironment_importBasic(t *testing.T) {
	resourceName := "aws_elastic_beanstalk_application.tftest"

	applicationName := fmt.Sprintf("tf-test-name-%d", acctest.RandInt())
	environmentName := fmt.Sprintf("tf-test-env-name-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnvImportConfig(applicationName, environmentName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccBeanstalkEnvImportConfig(appName, envName string) string {
	return fmt.Sprintf(`resource "aws_elastic_beanstalk_application" "tftest" {
	  name = "%s"
	  description = "tf-test-desc"
	}

	resource "aws_elastic_beanstalk_environment" "tfenvtest" {
	  name = "%s"
	  application = "${aws_elastic_beanstalk_application.tftest.name}"
	  solution_stack_name = "64bit Amazon Linux running Python"
	}`, appName, envName)
}
