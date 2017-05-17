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

func TestAccAWSSSMActivation_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMActivationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSSMActivationBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMActivationExists("aws_ssm_activation.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMActivationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Activation ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		_, err := conn.DescribeActivations(&ssm.DescribeActivationsInput{
			Filters: []*ssm.DescribeActivationsFilter{
				{
					FilterKey: aws.String("ActivationIds"),
					FilterValues: []*string{
						aws.String(rs.Primary.ID),
					},
				},
			},
			MaxResults: aws.Int64(1),
		})

		if err != nil {
			return fmt.Errorf("Could not descripbe the activation - %s", err)
		}

		return nil
	}
}

func testAccCheckAWSSSMActivationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_activation" {
			continue
		}

		out, err := conn.DescribeActivations(&ssm.DescribeActivationsInput{
			Filters: []*ssm.DescribeActivationsFilter{
				{
					FilterKey: aws.String("ActivationIds"),
					FilterValues: []*string{
						aws.String(rs.Primary.ID),
					},
				},
			},
			MaxResults: aws.Int64(1),
		})

		if err != nil {
			return err
		}

		if len(out.ActivationList) > 0 {
			return fmt.Errorf("Expected AWS SSM Activation to be gone, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in SSM Activation Test")
}

func testAccAWSSSMActivationBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test_role" {
  name = "test_role-%s"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ssm.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "test_attach" {
  role = "${aws_iam_role.test_role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforSSM"
}

resource "aws_ssm_activation" "foo" {
  name               = "test_ssm_activation-%s",
  description        = "Test"
  iam_role           = "${aws_iam_role.test_role.name}"
  registration_limit = "5"
  depends_on         = ["aws_iam_role_policy_attachment.test_attach"]
}
`, rName, rName)
}
