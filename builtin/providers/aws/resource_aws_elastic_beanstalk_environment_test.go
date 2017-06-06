package aws

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sort"
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
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnvConfig(rInt),
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
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkWorkerEnvConfig(rInt),
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
	rInt := acctest.RandInt()
	beanstalkAsgNameRegexp := regexp.MustCompile("awseb.+?AutoScalingGroup[^,]+")
	beanstalkElbNameRegexp := regexp.MustCompile("awseb.+?EBLoa[^,]+")
	beanstalkInstancesNameRegexp := regexp.MustCompile("i-([0-9a-fA-F]{8}|[0-9a-fA-F]{17})")
	beanstalkLcNameRegexp := regexp.MustCompile("awseb.+?AutoScalingLaunch[^,]+")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnvConfig(rInt),
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
	rInt := acctest.RandInt()
	beanstalkCnameRegexp := regexp.MustCompile("^" + cnamePrefix + ".+?elasticbeanstalk.com$")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnvCnamePrefixConfig(cnamePrefix, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					resource.TestMatchResourceAttr(
						"aws_elastic_beanstalk_environment.tfenvtest", "cname", beanstalkCnameRegexp),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_config(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkConfigTemplate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tftest", &app),
					testAccCheckBeanstalkEnvConfigValue("aws_elastic_beanstalk_environment.tftest", "1"),
				),
			},

			{
				Config: testAccBeanstalkConfigTemplateUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tftest", &app),
					testAccCheckBeanstalkEnvConfigValue("aws_elastic_beanstalk_environment.tftest", "2"),
				),
			},

			{
				Config: testAccBeanstalkConfigTemplateUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tftest", &app),
					testAccCheckBeanstalkEnvConfigValue("aws_elastic_beanstalk_environment.tftest", "3"),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_resource(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkResourceOptionSetting(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_vpc(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnv_VPC(acctest.RandString(5), rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.default", &app),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_template_change(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnv_TemplateChange_stack(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.environment", &app),
				),
			},
			{
				Config: testAccBeanstalkEnv_TemplateChange_temp(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.environment", &app),
				),
			},
			{
				Config: testAccBeanstalkEnv_TemplateChange_stack(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.environment", &app),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_basic_settings_update(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBeanstalkEnvConfig_empty_settings(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					testAccVerifyBeanstalkConfig(&app, []string{}),
				),
			},
			{
				Config: testAccBeanstalkEnvConfig_settings(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					testAccVerifyBeanstalkConfig(&app, []string{"ENV_STATIC", "ENV_UPDATE"}),
				),
			},
			{
				Config: testAccBeanstalkEnvConfig_settings_update(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					testAccVerifyBeanstalkConfig(&app, []string{"ENV_STATIC", "ENV_UPDATE"}),
				),
			},
			{
				Config: testAccBeanstalkEnvConfig_empty_settings(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkEnvExists("aws_elastic_beanstalk_environment.tfenvtest", &app),
					testAccVerifyBeanstalkConfig(&app, []string{}),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkEnv_version_label(t *testing.T) {
	var app elasticbeanstalk.EnvironmentDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkEnvDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkEnvApplicationVersionConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkApplicationVersionDeployed("aws_elastic_beanstalk_environment.default", &app),
				),
			},
			resource.TestStep{
				Config: testAccBeanstalkEnvApplicationVersionConfigUpdate(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkApplicationVersionDeployed("aws_elastic_beanstalk_environment.default", &app),
				),
			},
		},
	})
}

func testAccVerifyBeanstalkConfig(env *elasticbeanstalk.EnvironmentDescription, expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if env == nil {
			return fmt.Errorf("Nil environment in testAccVerifyBeanstalkConfig")
		}
		conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn

		resp, err := conn.DescribeConfigurationSettings(&elasticbeanstalk.DescribeConfigurationSettingsInput{
			ApplicationName: env.ApplicationName,
			EnvironmentName: env.EnvironmentName,
		})

		if err != nil {
			return fmt.Errorf("Error describing config settings in testAccVerifyBeanstalkConfig: %s", err)
		}

		// should only be 1 environment
		if len(resp.ConfigurationSettings) != 1 {
			return fmt.Errorf("Expected only 1 set of Configuration Settings in testAccVerifyBeanstalkConfig, got (%d)", len(resp.ConfigurationSettings))
		}

		cs := resp.ConfigurationSettings[0]

		var foundEnvs []string
		testStrings := []string{"ENV_STATIC", "ENV_UPDATE"}
		for _, os := range cs.OptionSettings {
			for _, k := range testStrings {
				if *os.OptionName == k {
					foundEnvs = append(foundEnvs, k)
				}
			}
		}

		// if expected is zero, then we should not have found any of the predefined
		// env vars
		if len(expected) == 0 {
			if len(foundEnvs) > 0 {
				return fmt.Errorf("Found configs we should not have: %#v", foundEnvs)
			}
			return nil
		}

		sort.Strings(testStrings)
		sort.Strings(expected)
		if !reflect.DeepEqual(testStrings, expected) {
			return fmt.Errorf("Error matching strings, expected:\n\n%#v\n\ngot:\n\n%#v\n", testStrings, foundEnvs)
		}

		return nil
	}
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

func testAccCheckBeanstalkEnvConfigValue(n string, expectedValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Elastic Beanstalk ENV is not set")
		}

		resp, err := conn.DescribeConfigurationOptions(&elasticbeanstalk.DescribeConfigurationOptionsInput{
			ApplicationName: aws.String(rs.Primary.Attributes["application"]),
			EnvironmentName: aws.String(rs.Primary.Attributes["name"]),
			Options: []*elasticbeanstalk.OptionSpecification{
				{
					Namespace:  aws.String("aws:elasticbeanstalk:application:environment"),
					OptionName: aws.String("TEMPLATE"),
				},
			},
		})
		if err != nil {
			return err
		}

		if len(resp.Options) != 1 {
			return fmt.Errorf("Found %d options, expected 1.", len(resp.Options))
		}

		log.Printf("[DEBUG] %d Elastic Beanstalk Option values returned.", len(resp.Options[0].ValueOptions))

		for _, value := range resp.Options[0].ValueOptions {
			if *value != expectedValue {
				return fmt.Errorf("Option setting value: %s. Expected %s", *value, expectedValue)
			}
		}

		return nil
	}
}

func testAccCheckBeanstalkApplicationVersionDeployed(n string, app *elasticbeanstalk.EnvironmentDescription) resource.TestCheckFunc {
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

		if *env.VersionLabel != rs.Primary.Attributes["version_label"] {
			return fmt.Errorf("Elastic Beanstalk version deployed %s. Expected %s", *env.VersionLabel, rs.Primary.Attributes["version_label"])
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

func testAccBeanstalkEnvConfig(rInt int) string {
	return fmt.Sprintf(`
 resource "aws_elastic_beanstalk_application" "tftest" {
	 name = "tf-test-name-%d"
	 description = "tf-test-desc"
 }

 resource "aws_elastic_beanstalk_environment" "tfenvtest" {
	 name = "tf-test-name-%d"
	 application = "${aws_elastic_beanstalk_application.tftest.name}"
	 solution_stack_name = "64bit Amazon Linux running Python"
	 depends_on = ["aws_elastic_beanstalk_application.tftest"]
 }
 `, rInt, rInt)
}

func testAccBeanstalkEnvConfig_empty_settings(r int) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name = "tf-test-name-%d"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux running Python"

        wait_for_ready_timeout = "15m"
}`, r, r)
}

func testAccBeanstalkEnvConfig_settings(r int) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name                = "tf-test-name-%d"
  application         = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux running Python"

        wait_for_ready_timeout = "15m"

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "ENV_STATIC"
    value     = "true"
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "ENV_UPDATE"
    value     = "true"
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "ENV_REMOVE"
    value     = "true"
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource  = "ScheduledAction01"
    name      = "MinSize"
    value     = 2
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource  = "ScheduledAction01"
    name      = "MaxSize"
    value     = 3
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource  = "ScheduledAction01"
    name      = "StartTime"
    value     = "2016-07-28T04:07:02Z"
  }
}`, r, r)
}

func testAccBeanstalkEnvConfig_settings_update(r int) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name                = "tf-test-name-%d"
  application         = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux running Python"

        wait_for_ready_timeout = "15m"

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "ENV_STATIC"
    value     = "true"
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "ENV_UPDATE"
    value     = "false"
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "ENV_ADD"
    value     = "true"
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource  = "ScheduledAction01"
    name      = "MinSize"
    value     = 2
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource  = "ScheduledAction01"
    name      = "MaxSize"
    value     = 3
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource  = "ScheduledAction01"
    name      = "StartTime"
    value     = "2016-07-28T04:07:02Z"
  }
}`, r, r)
}

func testAccBeanstalkWorkerEnvConfig(rInt int) string {
	return fmt.Sprintf(`
 resource "aws_iam_instance_profile" "tftest" {
	 name = "tftest_profile-%d"
	 roles = ["${aws_iam_role.tftest.name}"]
 }

 resource "aws_iam_role" "tftest" {
	 name = "tftest_role"
	 path = "/"
	 assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Effect\":\"Allow\",\"Sid\":\"\"}]}"
 }

 resource "aws_iam_role_policy" "tftest" {
	 name = "tftest_policy"
	 role = "${aws_iam_role.tftest.id}"
	 policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"QueueAccess\",\"Action\":[\"sqs:ChangeMessageVisibility\",\"sqs:DeleteMessage\",\"sqs:ReceiveMessage\"],\"Effect\":\"Allow\",\"Resource\":\"*\"}]}"
 }

 resource "aws_elastic_beanstalk_application" "tftest" {
	 name = "tf-test-name-%d"
	 description = "tf-test-desc"
 }

 resource "aws_elastic_beanstalk_environment" "tfenvtest" {
	 name = "tf-test-name-%d"
	 application = "${aws_elastic_beanstalk_application.tftest.name}"
	 tier = "Worker"
	 solution_stack_name = "64bit Amazon Linux running Python"

	 setting {
		 namespace = "aws:autoscaling:launchconfiguration"
		 name      = "IamInstanceProfile"
		 value     = "${aws_iam_instance_profile.tftest.name}"
	 }
 }`, rInt, rInt, rInt)
}

func testAccBeanstalkEnvCnamePrefixConfig(randString string, rInt int) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
name = "tf-test-name-%d"
description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
name = "tf-test-name-%d"
application = "${aws_elastic_beanstalk_application.tftest.name}"
cname_prefix = "%s"
solution_stack_name = "64bit Amazon Linux running Python"
}
`, rInt, rInt, randString)
}

func testAccBeanstalkConfigTemplate(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_elastic_beanstalk_application" "tftest" {
		name = "tf-test-name-%d"
		description = "tf-test-desc"
	}

	resource "aws_elastic_beanstalk_environment" "tftest" {
		name = "tf-test-name-%d"
		application = "${aws_elastic_beanstalk_application.tftest.name}"
		template_name = "${aws_elastic_beanstalk_configuration_template.tftest.name}"
	}

	resource "aws_elastic_beanstalk_configuration_template" "tftest" {
		name        = "tf-test-original"
		application = "${aws_elastic_beanstalk_application.tftest.name}"
		solution_stack_name = "64bit Amazon Linux running Python"

		setting {
			namespace = "aws:elasticbeanstalk:application:environment"
			name      = "TEMPLATE"
			value     = "1"
	 }
	}
	`, rInt, rInt)
}

func testAccBeanstalkConfigTemplateUpdate(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_elastic_beanstalk_application" "tftest" {
		name = "tf-test-name-%d"
		description = "tf-test-desc"
	}

	resource "aws_elastic_beanstalk_environment" "tftest" {
		name = "tf-test-name-%d"
		application = "${aws_elastic_beanstalk_application.tftest.name}"
		template_name = "${aws_elastic_beanstalk_configuration_template.tftest.name}"
	}

	resource "aws_elastic_beanstalk_configuration_template" "tftest" {
		name        = "tf-test-updated"
		application = "${aws_elastic_beanstalk_application.tftest.name}"
		solution_stack_name = "64bit Amazon Linux running Python"

		setting {
			namespace = "aws:elasticbeanstalk:application:environment"
			name      = "TEMPLATE"
			value     = "2"
		}
	}
	`, rInt, rInt)
}

func testAccBeanstalkResourceOptionSetting(rInt int) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name = "tf-test-name-%d"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux running Python"

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource = "ScheduledAction01"
    name = "MinSize"
    value = "2"
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource = "ScheduledAction01"
    name = "MaxSize"
    value = "6"
  }

  setting {
    namespace = "aws:autoscaling:scheduledaction"
    resource = "ScheduledAction01"
    name = "Recurrence"
    value = "0 8 * * *"
  }
}`, rInt, rInt)
}

func testAccBeanstalkEnv_VPC(name string, rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "tf_b_test" {
  cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccBeanstalkEnv_VPC"
	}
}

resource "aws_internet_gateway" "tf_b_test" {
  vpc_id = "${aws_vpc.tf_b_test.id}"
}

resource "aws_route" "r" {
  route_table_id = "${aws_vpc.tf_b_test.main_route_table_id}"
  destination_cidr_block = "0.0.0.0/0"
  gateway_id = "${aws_internet_gateway.tf_b_test.id}"
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.tf_b_test.id}"
  cidr_block = "10.0.0.0/24"
}

resource "aws_security_group" "default" {
  name = "tf-b-test-%s"
  vpc_id = "${aws_vpc.tf_b_test.id}"
}

resource "aws_elastic_beanstalk_application" "default" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "default" {
  name = "tf-test-name-%d"
  application = "${aws_elastic_beanstalk_application.default.name}"
  solution_stack_name = "64bit Amazon Linux running Python"

  setting {
    namespace = "aws:ec2:vpc"
    name      = "VPCId"
    value     = "${aws_vpc.tf_b_test.id}"
  }

  setting {
    namespace = "aws:ec2:vpc"
    name      = "Subnets"
    value     = "${aws_subnet.main.id}"
  }

  setting {
    namespace = "aws:ec2:vpc"
    name      = "AssociatePublicIpAddress"
    value     = "true"
  }

  setting {
    namespace = "aws:autoscaling:launchconfiguration"
    name      = "SecurityGroups"
    value     = "${aws_security_group.default.id}"
  }
}
`, name, rInt, rInt)
}

func testAccBeanstalkEnv_TemplateChange_stack(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}

resource "aws_elastic_beanstalk_application" "app" {
  name        = "beanstalk-app-%d"
  description = ""
}

resource "aws_elastic_beanstalk_environment" "environment" {
  name        = "beanstalk-env-%d"
  application = "${aws_elastic_beanstalk_application.app.name}"

  # Go 1.4

  solution_stack_name = "64bit Amazon Linux 2016.03 v2.1.0 running Go 1.4"
}

resource "aws_elastic_beanstalk_configuration_template" "template" {
  name        = "beanstalk-config-%d"
  application = "${aws_elastic_beanstalk_application.app.name}"

  # Go 1.5
  solution_stack_name = "64bit Amazon Linux 2016.03 v2.1.3 running Go 1.5"
}
`, r, r, r)
}

func testAccBeanstalkEnv_TemplateChange_temp(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}

resource "aws_elastic_beanstalk_application" "app" {
  name        = "beanstalk-app-%d"
  description = ""
}

resource "aws_elastic_beanstalk_environment" "environment" {
  name        = "beanstalk-env-%d"
  application = "${aws_elastic_beanstalk_application.app.name}"

  # Go 1.4

  template_name = "${aws_elastic_beanstalk_configuration_template.template.name}"
}

resource "aws_elastic_beanstalk_configuration_template" "template" {
  name        = "beanstalk-config-%d"
  application = "${aws_elastic_beanstalk_application.app.name}"

  # Go 1.5
  solution_stack_name = "64bit Amazon Linux 2016.03 v2.1.3 running Go 1.5"
}
`, r, r, r)
}

func testAccBeanstalkEnvApplicationVersionConfig(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "default" {
  bucket = "tftest.applicationversion.buckets-%d"
}

resource "aws_s3_bucket_object" "default" {
  bucket = "${aws_s3_bucket.default.id}"
  key = "python-v1.zip"
  source = "test-fixtures/python-v1.zip"
}

resource "aws_elastic_beanstalk_application" "default" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_application_version" "default" {
  application = "${aws_elastic_beanstalk_application.default.name}"
  name = "tf-test-version-label"
  bucket = "${aws_s3_bucket.default.id}"
  key = "${aws_s3_bucket_object.default.id}"
}

resource "aws_elastic_beanstalk_environment" "default" {
  name = "tf-test-name-%d"
  application = "${aws_elastic_beanstalk_application.default.name}"
  version_label = "${aws_elastic_beanstalk_application_version.default.name}"
  solution_stack_name = "64bit Amazon Linux running Python"
}
`, randInt, randInt, randInt)
}

func testAccBeanstalkEnvApplicationVersionConfigUpdate(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "default" {
  bucket = "tftest.applicationversion.buckets-%d"
}

resource "aws_s3_bucket_object" "default" {
  bucket = "${aws_s3_bucket.default.id}"
  key = "python-v2.zip"
  source = "test-fixtures/python-v1.zip"
}

resource "aws_elastic_beanstalk_application" "default" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_application_version" "default" {
  application = "${aws_elastic_beanstalk_application.default.name}"
  name = "tf-test-version-label-v2"
  bucket = "${aws_s3_bucket.default.id}"
  key = "${aws_s3_bucket_object.default.id}"
}

resource "aws_elastic_beanstalk_environment" "default" {
  name = "tf-test-name-%d"
  application = "${aws_elastic_beanstalk_application.default.name}"
  version_label = "${aws_elastic_beanstalk_application_version.default.name}"
  solution_stack_name = "64bit Amazon Linux running Python"
}
`, randInt, randInt, randInt)
}
