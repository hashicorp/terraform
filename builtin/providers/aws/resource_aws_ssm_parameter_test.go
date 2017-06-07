package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMParameter_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterBasicConfig(name, "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterHasValue("aws_ssm_parameter.foo", "bar"),
					testAccCheckAWSSSMParameterType("aws_ssm_parameter.foo", "String"),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_update(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterBasicConfig(name, "bar"),
			},
			{
				Config: testAccAWSSSMParameterBasicConfig(name, "baz"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterHasValue("aws_ssm_parameter.foo", "baz"),
					testAccCheckAWSSSMParameterType("aws_ssm_parameter.foo", "String"),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_secure(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterSecureConfig(name, "secret"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterHasValue("aws_ssm_parameter.secret_foo", "secret"),
					testAccCheckAWSSSMParameterType("aws_ssm_parameter.secret_foo", "SecureString"),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_secure_with_key(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterSecureConfigWithKey(name, "secret"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterHasValue("aws_ssm_parameter.secret_foo", "secret"),
					testAccCheckAWSSSMParameterType("aws_ssm_parameter.secret_foo", "SecureString"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMGetParameter(s *terraform.State, n string) ([]*ssm.Parameter, error) {
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return []*ssm.Parameter{}, fmt.Errorf("Not found: %s", n)
	}

	if rs.Primary.ID == "" {
		return []*ssm.Parameter{}, fmt.Errorf("No SSM Parameter ID is set")
	}

	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	paramInput := &ssm.GetParametersInput{
		Names: []*string{
			aws.String(rs.Primary.Attributes["name"]),
		},
		WithDecryption: aws.Bool(true),
	}

	resp, _ := conn.GetParameters(paramInput)

	if len(resp.Parameters) == 0 {
		return resp.Parameters, fmt.Errorf("Expected AWS SSM Parameter to be created, but wasn't found")
	}
	return resp.Parameters, nil
}

func testAccCheckAWSSSMParameterHasValue(n string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		parameters, err := testAccCheckAWSSSMGetParameter(s, n)
		if err != nil {
			return err
		}

		parameterValue := parameters[0].Value

		if *parameterValue != v {
			return fmt.Errorf("Expected AWS SSM Parameter to have value %s but had %s", v, *parameterValue)
		}

		return nil
	}
}

func testAccCheckAWSSSMParameterType(n string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		parameters, err := testAccCheckAWSSSMGetParameter(s, n)
		if err != nil {
			return err
		}

		parameterValue := parameters[0].Type

		if *parameterValue != v {
			return fmt.Errorf("Expected AWS SSM Parameter to have type %s but had %s", v, *parameterValue)
		}

		return nil
	}
}

func testAccCheckAWSSSMParameterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_parameter" {
			continue
		}

		paramInput := &ssm.GetParametersInput{
			Names: []*string{
				aws.String(rs.Primary.Attributes["name"]),
			},
		}

		resp, _ := conn.GetParameters(paramInput)

		if len(resp.Parameters) > 0 {
			return fmt.Errorf("Expected AWS SSM Parameter to be gone, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in SSM Parameter Test")
}

func testAccAWSSSMParameterBasicConfig(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "foo" {
  name  = "test_parameter-%s"
  type  = "String"
  value = "%s"
}
`, rName, value)
}

func testAccAWSSSMParameterSecureConfig(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "secret_foo" {
  name  = "test_secure_parameter-%s"
  type  = "SecureString"
  value = "%s"
}
`, rName, value)
}

func testAccAWSSSMParameterSecureConfigWithKey(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "secret_foo" {
  name  = "test_secure_parameter-%s"
  type  = "SecureString"
  value = "%s"
	key_id = "${aws_kms_key.test_key.id}"
}

resource "aws_kms_key" "test_key" {
  description             = "KMS key 1"
  deletion_window_in_days = 7
}
`, rName, value)
}
