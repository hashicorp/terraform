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

func testAccConfigConfigRule_basic(t *testing.T) {
	var cr configservice.ConfigRule
	rInt := acctest.RandInt()
	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigRuleConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigRuleExists("aws_config_config_rule.foo", &cr),
					testAccCheckConfigConfigRuleName("aws_config_config_rule.foo", expectedName, &cr),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "name", expectedName),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.owner", "AWS"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_identifier", "S3_BUCKET_VERSIONING_ENABLED"),
				),
			},
		},
	})
}

func testAccConfigConfigRule_ownerAws(t *testing.T) {
	var cr configservice.ConfigRule
	rInt := acctest.RandInt()
	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)
	expectedArn := regexp.MustCompile("arn:aws:config:[a-z0-9-]+:[0-9]{12}:config-rule/config-rule-([a-z0-9]+)")
	expectedRuleId := regexp.MustCompile("config-rule-[a-z0-9]+")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigRuleConfig_ownerAws(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigRuleExists("aws_config_config_rule.foo", &cr),
					testAccCheckConfigConfigRuleName("aws_config_config_rule.foo", expectedName, &cr),
					resource.TestMatchResourceAttr("aws_config_config_rule.foo", "arn", expectedArn),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "name", expectedName),
					resource.TestMatchResourceAttr("aws_config_config_rule.foo", "rule_id", expectedRuleId),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "description", "Terraform Acceptance tests"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.owner", "AWS"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_identifier", "REQUIRED_TAGS"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_detail.#", "0"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.0.compliance_resource_id", "blablah"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.0.compliance_resource_types.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.0.compliance_resource_types.3865728585", "AWS::EC2::Instance"),
				),
			},
		},
	})
}

func testAccConfigConfigRule_customlambda(t *testing.T) {
	var cr configservice.ConfigRule
	rInt := acctest.RandInt()

	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)
	path := "test-fixtures/lambdatest.zip"
	expectedArn := regexp.MustCompile("arn:aws:config:[a-z0-9-]+:[0-9]{12}:config-rule/config-rule-([a-z0-9]+)")
	expectedFunctionArn := regexp.MustCompile(fmt.Sprintf("arn:aws:lambda:[a-z0-9-]+:[0-9]{12}:function:tf_acc_lambda_awsconfig_%d", rInt))
	expectedRuleId := regexp.MustCompile("config-rule-[a-z0-9]+")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigRuleConfig_customLambda(rInt, path),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigRuleExists("aws_config_config_rule.foo", &cr),
					testAccCheckConfigConfigRuleName("aws_config_config_rule.foo", expectedName, &cr),
					resource.TestMatchResourceAttr("aws_config_config_rule.foo", "arn", expectedArn),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "name", expectedName),
					resource.TestMatchResourceAttr("aws_config_config_rule.foo", "rule_id", expectedRuleId),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "description", "Terraform Acceptance tests"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "maximum_execution_frequency", "Six_Hours"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.owner", "CUSTOM_LAMBDA"),
					resource.TestMatchResourceAttr("aws_config_config_rule.foo", "source.0.source_identifier", expectedFunctionArn),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_detail.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_detail.3026922761.event_source", "aws.config"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_detail.3026922761.message_type", "ConfigurationSnapshotDeliveryCompleted"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "source.0.source_detail.3026922761.maximum_execution_frequency", ""),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.#", "1"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.0.tag_key", "IsTemporary"),
					resource.TestCheckResourceAttr("aws_config_config_rule.foo", "scope.0.tag_value", "yes"),
				),
			},
		},
	})
}

func testAccConfigConfigRule_importAws(t *testing.T) {
	resourceName := "aws_config_config_rule.foo"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccConfigConfigRuleConfig_ownerAws(rInt),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccConfigConfigRule_importLambda(t *testing.T) {
	resourceName := "aws_config_config_rule.foo"
	rInt := acctest.RandInt()

	path := "test-fixtures/lambdatest.zip"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccConfigConfigRuleConfig_customLambda(rInt, path),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckConfigConfigRuleName(n, desired string, obj *configservice.ConfigRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != *obj.ConfigRuleName {
			return fmt.Errorf("Expected name: %q, given: %q", desired, *obj.ConfigRuleName)
		}
		return nil
	}
}

func testAccCheckConfigConfigRuleExists(n string, obj *configservice.ConfigRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No config rule ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).configconn
		out, err := conn.DescribeConfigRules(&configservice.DescribeConfigRulesInput{
			ConfigRuleNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})
		if err != nil {
			return fmt.Errorf("Failed to describe config rule: %s", err)
		}
		if len(out.ConfigRules) < 1 {
			return fmt.Errorf("No config rule found when describing %q", rs.Primary.Attributes["name"])
		}

		cr := out.ConfigRules[0]
		*obj = *cr

		return nil
	}
}

func testAccCheckConfigConfigRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).configconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_config_config_rule" {
			continue
		}

		resp, err := conn.DescribeConfigRules(&configservice.DescribeConfigRulesInput{
			ConfigRuleNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})

		if err == nil {
			if len(resp.ConfigRules) != 0 &&
				*resp.ConfigRules[0].ConfigRuleName == rs.Primary.Attributes["name"] {
				return fmt.Errorf("config rule still exists: %s", rs.Primary.Attributes["name"])
			}
		}
	}

	return nil
}

func testAccConfigConfigRuleConfig_basic(randInt int) string {
	return fmt.Sprintf(`
resource "aws_config_config_rule" "foo" {
    name = "tf-acc-test-%d"
    source {
        owner = "AWS"
        source_identifier = "S3_BUCKET_VERSIONING_ENABLED"
    }
    depends_on = ["aws_config_configuration_recorder.foo"]
}

resource "aws_config_configuration_recorder" "foo" {
  name = "tf-acc-test-%d"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
    name = "tf-acc-test-awsconfig-%d"
    assume_role_policy = <<EOF
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
EOF
}

resource "aws_iam_role_policy" "p" {
    name = "tf-acc-test-awsconfig-%d"
    role = "${aws_iam_role.r.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
        "Action": "config:Put*",
        "Effect": "Allow",
        "Resource": "*"

    }
  ]
}
EOF
}`, randInt, randInt, randInt, randInt)
}

func testAccConfigConfigRuleConfig_ownerAws(randInt int) string {
	return fmt.Sprintf(`
resource "aws_config_config_rule" "foo" {
    name = "tf-acc-test-%d"
    description = "Terraform Acceptance tests"
    source {
        owner = "AWS"
        source_identifier = "REQUIRED_TAGS"
    }
    scope {
    	compliance_resource_id = "blablah"
    	compliance_resource_types = ["AWS::EC2::Instance"]
    }
    input_parameters = <<PARAMS
{"tag1Key":"CostCenter", "tag2Key":"Owner"}
PARAMS
    depends_on = ["aws_config_configuration_recorder.foo"]
}

resource "aws_config_configuration_recorder" "foo" {
  name = "tf-acc-test-%d"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
    name = "tf-acc-test-awsconfig-%d"
    assume_role_policy = <<EOF
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
EOF
}

resource "aws_iam_role_policy" "p" {
    name = "tf-acc-test-awsconfig-%d"
    role = "${aws_iam_role.r.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
        "Action": "config:Put*",
        "Effect": "Allow",
        "Resource": "*"

    }
  ]
}
EOF
}`, randInt, randInt, randInt, randInt)
}

func testAccConfigConfigRuleConfig_customLambda(randInt int, path string) string {
	return fmt.Sprintf(`
resource "aws_config_config_rule" "foo" {
  name = "tf-acc-test-%d"
  description = "Terraform Acceptance tests"
  maximum_execution_frequency = "Six_Hours"
  source {
      owner = "CUSTOM_LAMBDA"
      source_identifier = "${aws_lambda_function.f.arn}"
      source_detail {
        event_source = "aws.config"
        message_type = "ConfigurationSnapshotDeliveryCompleted"
      }
  }
  scope {
    tag_key = "IsTemporary"
    tag_value = "yes"
  }
  depends_on = [
    "aws_config_configuration_recorder.foo",
    "aws_config_delivery_channel.foo",
  ]
}

resource "aws_lambda_function" "f" {
  filename = "%s"
  function_name = "tf_acc_lambda_awsconfig_%d"
  role = "${aws_iam_role.iam_for_lambda.arn}"
  handler = "exports.example"
  runtime = "nodejs4.3"
}

resource "aws_lambda_permission" "p" {
  statement_id = "AllowExecutionFromConfig"
  action = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.f.arn}"
  principal = "config.amazonaws.com"
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "tf_acc_lambda_aws_config_%d"
  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy_attachment" "a" {
  role = "${aws_iam_role.iam_for_lambda.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSConfigRulesExecutionRole"
}

resource "aws_config_delivery_channel" "foo" {
  name = "tf-acc-test-%d"
  s3_bucket_name = "${aws_s3_bucket.b.bucket}"
  snapshot_delivery_properties {
    delivery_frequency = "Six_Hours"
  }
  depends_on = ["aws_config_configuration_recorder.foo"]
}

resource "aws_s3_bucket" "b" {
  bucket = "tf-acc-awsconfig-%d"
  force_destroy = true
}

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
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "config:Put*",
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": "s3:*",
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.b.arn}",
        "${aws_s3_bucket.b.arn}/*"
      ]
    },
    {
      "Action": "lambda:*",
      "Effect": "Allow",
      "Resource": "${aws_lambda_function.f.arn}"
    }
  ]
}
POLICY
}`, randInt, path, randInt, randInt, randInt, randInt, randInt, randInt, randInt)
}
