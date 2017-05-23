package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func testAccConfigConfigurationRecorder_basic(t *testing.T) {
	var cr configservice.ConfigurationRecorder
	rInt := acctest.RandInt()
	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)
	expectedRoleName := fmt.Sprintf("tf-acc-test-awsconfig-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigurationRecorderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigurationRecorderConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigurationRecorderExists("aws_config_configuration_recorder.foo", &cr),
					testAccCheckConfigConfigurationRecorderName("aws_config_configuration_recorder.foo", expectedName, &cr),
					testAccCheckConfigConfigurationRecorderRoleArn("aws_config_configuration_recorder.foo",
						regexp.MustCompile(`arn:aws:iam::[0-9]{12}:role/`+expectedRoleName), &cr),
					resource.TestCheckResourceAttr("aws_config_configuration_recorder.foo", "name", expectedName),
					resource.TestMatchResourceAttr("aws_config_configuration_recorder.foo", "role_arn",
						regexp.MustCompile(`arn:aws:iam::[0-9]{12}:role/`+expectedRoleName)),
				),
			},
		},
	})
}

func testAccConfigConfigurationRecorder_allParams(t *testing.T) {
	var cr configservice.ConfigurationRecorder
	rInt := acctest.RandInt()
	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)
	expectedRoleName := fmt.Sprintf("tf-acc-test-awsconfig-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigurationRecorderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigurationRecorderConfig_allParams(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigurationRecorderExists("aws_config_configuration_recorder.foo", &cr),
					testAccCheckConfigConfigurationRecorderName("aws_config_configuration_recorder.foo", expectedName, &cr),
					testAccCheckConfigConfigurationRecorderRoleArn("aws_config_configuration_recorder.foo",
						regexp.MustCompile(`arn:aws:iam::[0-9]{12}:role/`+expectedRoleName), &cr),
					resource.TestCheckResourceAttr("aws_config_configuration_recorder.foo", "name", expectedName),
					resource.TestMatchResourceAttr("aws_config_configuration_recorder.foo", "role_arn",
						regexp.MustCompile(`arn:aws:iam::[0-9]{12}:role/`+expectedRoleName)),
					resource.TestCheckResourceAttr("aws_config_configuration_recorder.foo", "recording_group.#", "1"),
					resource.TestCheckResourceAttr("aws_config_configuration_recorder.foo", "recording_group.0.all_supported", "false"),
					resource.TestCheckResourceAttr("aws_config_configuration_recorder.foo", "recording_group.0.include_global_resource_types", "false"),
					resource.TestCheckResourceAttr("aws_config_configuration_recorder.foo", "recording_group.0.resource_types.#", "2"),
				),
			},
		},
	})
}

func testAccConfigConfigurationRecorder_importBasic(t *testing.T) {
	resourceName := "aws_config_configuration_recorder.foo"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigurationRecorderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccConfigConfigurationRecorderConfig_basic(rInt),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckConfigConfigurationRecorderName(n string, desired string, obj *configservice.ConfigurationRecorder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if *obj.Name != desired {
			return fmt.Errorf("Expected configuration recorder %q name to be %q, given: %q",
				n, desired, *obj.Name)
		}

		return nil
	}
}

func testAccCheckConfigConfigurationRecorderRoleArn(n string, desired *regexp.Regexp, obj *configservice.ConfigurationRecorder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if !desired.MatchString(*obj.RoleARN) {
			return fmt.Errorf("Expected configuration recorder %q role ARN to match %q, given: %q",
				n, desired.String(), *obj.RoleARN)
		}

		return nil
	}
}

func testAccCheckConfigConfigurationRecorderExists(n string, obj *configservice.ConfigurationRecorder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No configuration recorder ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).configconn
		out, err := conn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})
		if err != nil {
			return fmt.Errorf("Failed to describe configuration recorder: %s", err)
		}
		if len(out.ConfigurationRecorders) < 1 {
			return fmt.Errorf("No configuration recorder found when describing %q", rs.Primary.Attributes["name"])
		}

		cr := out.ConfigurationRecorders[0]
		*obj = *cr

		return nil
	}
}

func testAccCheckConfigConfigurationRecorderDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).configconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_config_configuration_recorder_status" {
			continue
		}

		resp, err := conn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})

		if err == nil {
			if len(resp.ConfigurationRecorders) != 0 &&
				*resp.ConfigurationRecorders[0].Name == rs.Primary.Attributes["name"] {
				return fmt.Errorf("Configuration recorder still exists: %s", rs.Primary.Attributes["name"])
			}
		}
	}

	return nil
}

func testAccConfigConfigurationRecorderConfig_basic(randInt int) string {
	return fmt.Sprintf(`
resource "aws_config_configuration_recorder" "foo" {
  name = "tf-acc-test-%d"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
    name = "tf-acc-test-awsconfig-%d"
    assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "p" {
    name = "tf-acc-test-awsconfig-%d"
    role = "${aws_iam_role.r.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:*"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.b.arn}",
        "${aws_s3_bucket.b.arn}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "b" {
  bucket = "tf-acc-test-awsconfig-%d"
  force_destroy = true
}

resource "aws_config_delivery_channel" "foo" {
  name = "tf-acc-test-awsconfig-%d"
  s3_bucket_name = "${aws_s3_bucket.b.bucket}"
  depends_on = ["aws_config_configuration_recorder.foo"]
}
`, randInt, randInt, randInt, randInt, randInt)
}

func testAccConfigConfigurationRecorderConfig_allParams(randInt int) string {
	return fmt.Sprintf(`
resource "aws_config_configuration_recorder" "foo" {
  name = "tf-acc-test-%d"
  role_arn = "${aws_iam_role.r.arn}"
  recording_group {
    all_supported = false
    include_global_resource_types = false
    resource_types = ["AWS::EC2::Instance", "AWS::CloudTrail::Trail"]
  }
}

resource "aws_iam_role" "r" {
    name = "tf-acc-test-awsconfig-%d"
    assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "p" {
    name = "tf-acc-test-awsconfig-%d"
    role = "${aws_iam_role.r.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:*"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.b.arn}",
        "${aws_s3_bucket.b.arn}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "b" {
  bucket = "tf-acc-test-awsconfig-%d"
  force_destroy = true
}

resource "aws_config_delivery_channel" "foo" {
  name = "tf-acc-test-awsconfig-%d"
  s3_bucket_name = "${aws_s3_bucket.b.bucket}"
  depends_on = ["aws_config_configuration_recorder.foo"]
}
`, randInt, randInt, randInt, randInt, randInt)
}
