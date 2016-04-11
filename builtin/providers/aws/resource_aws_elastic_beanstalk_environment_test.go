package aws

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSBeanstalkEnv_basic(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkEnvConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_tier(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription
	beanstalkQueuesNameRegexp := regexp.MustCompile("https://sqs.+?awseb[^,]+")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkWorkerEnvConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvTier("aws_elastic_beanstalk_environment.tfenvtest", &app),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "queues.0", beanstalkQueuesNameRegexp),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_outputs(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription
	beanstalkAsgNameRegexp := regexp.MustCompile("awseb.+?AutoScalingGroup[^,]+")
	beanstalkElbNameRegexp := regexp.MustCompile("awseb.+?EBLoa[^,]+")
	beanstalkInstancesNameRegexp := regexp.MustCompile("i-([0-9a-fA-F]{8}|[0-9a-fA-F]{17})")
	beanstalkLcNameRegexp := regexp.MustCompile("awseb.+?AutoScalingLaunch[^,]+")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkEnvConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "autoscaling_groups.0", beanstalkAsgNameRegexp),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "load_balancers.0", beanstalkElbNameRegexp),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "instances.0", beanstalkInstancesNameRegexp),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "launch_configurations.0", beanstalkLcNameRegexp),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_cname_prefix(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription
	cnamePrefix := acctest.RandString(8)
	beanstalkCnameRegexp := regexp.MustCompile("^" + cnamePrefix + ".+?elasticbeanstalk.com$")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkEnvCnamePrefixConfig(cnamePrefix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "cname", beanstalkCnameRegexp),
				),
			},
		},
	})
}

func testAccCheckBeanstalkEnvDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elastic_beanstalk_environment" {
			continue
		}

		// Try to find the environment
		describeBeanstalkEnvOpts := &elasticbeanstalk.DescribeEnvironmentsInput{
			EnvironmentIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeEnvironments(describeBeanstalkEnvOpts)
		if err == nil {
			switch {
			case len(resp.Environments) > 1:
				return fmt.Errorf("Error %d environments match, expected 1", len(resp.Environments))
			case len(resp.Environments) == 1:
				if *resp.Environments[0].Status == "Terminated" {
					return nil
				}
				return fmt.Errorf("Elastic Beanstalk ENV still exists.")
			default:
				return nil
			}
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidBeanstalkEnvID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckBeanstalkEnvExists(n string, app *elasticbeanstalk.EnvironmentDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Elastic Beanstalk ENV is not set")
		}

		env, err := describeBeanstalkEnv(testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn, aws.String(rs.Primary.ID))
		if err != nil {
			return err
		}

		*app = *env

		return nil
	}
}

func testAccCheckBeanstalkEnvTier(n string, app *elasticbeanstalk.EnvironmentDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Elastic Beanstalk ENV is not set")
		}

		env, err := describeBeanstalkEnv(testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn, aws.String(rs.Primary.ID))
		if err != nil {
			return err
		}
		if *env.Tier.Name != "Worker" {
			return fmt.Errorf("Beanstalk Environment tier is %s, expected Worker", *env.Tier.Name)
		}

		*app = *env

		return nil
	}
}

func describeBeanstalkEnv(conn *elasticbeanstalk.ElasticBeanstalk,
	envID *string) (*elasticbeanstalk.EnvironmentDescription, error) {
	describeBeanstalkEnvOpts := &elasticbeanstalk.DescribeEnvironmentsInput{
		EnvironmentIds: []*string{envID},
	}

	log.Printf("[DEBUG] Elastic Beanstalk Environment TEST describe opts: %s", describeBeanstalkEnvOpts)

	resp, err := conn.DescribeEnvironments(describeBeanstalkEnvOpts)
	if err != nil {
		return &elasticbeanstalk.EnvironmentDescription{}, err
	}
	if len(resp.Environments) == 0 {
		return &elasticbeanstalk.EnvironmentDescription{}, fmt.Errorf("Elastic Beanstalk ENV not found.")
	}
	if len(resp.Environments) > 1 {
		return &elasticbeanstalk.EnvironmentDescription{}, fmt.Errorf("Found %d environments, expected 1.", len(resp.Environments))
	}
	return resp.Environments[0], nil
}

const testAccBeanstalkEnvConfig = `
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name = "tf-test-name"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux running Python"
}
`

const testAccBeanstalkWorkerEnvConfig = `
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name = "tf-test-name"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  tier = "Worker"
  solution_stack_name = "64bit Amazon Linux running Python"
}
`

func testAccBeanstalkEnvCnamePrefixConfig(randString string) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
name = "tf-test-name"
description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
name = "tf-test-name"
application = "${aws_elastic_beanstalk_application.tftest.name}"
cname_prefix = "%s"
solution_stack_name = "64bit Amazon Linux running Python"
}
`, randString)
}
