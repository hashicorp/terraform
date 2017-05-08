package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codestar"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeStarProject_basic(t *testing.T) {
	name := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeStarProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeStarProjectConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeStarProjectExists("aws_codestar_project.foo"),
				),
			},
			{
				Config: testAccAWSCodeStarProjectConfig_basicUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeStarProjectExists("aws_codestar_project.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSCodeStarProjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No CodeStar Project ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).codestarconn

		out, err := conn.DescribeProject(&codestar.DescribeProjectInput{
			Id: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if aws.StringValue(out.Id) != rs.Primary.ID {
			return fmt.Errorf("No project found")
		}

		return nil
	}
}

func testAccCheckAWSCodeStarProjectDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codebuildconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codestar_project" {
			continue
		}

		out, err := conn.BatchGetProjects(&codebuild.BatchGetProjectsInput{
			Names: []*string{
				aws.String(rs.Primary.ID),
			},
		})

		if err != nil {
			return err
		}

		if out != nil && len(out.Projects) > 0 {
			return fmt.Errorf("Expected AWS CodeBuild Project to be gone, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in CodeBuild Test")
}

func testAccAWSCodeStarProjectConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "codestar_role" {
  name = "codestar-role-%s"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codestar.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "codestar_policy" {
    name        = "codestar-policy-%s"
    path        = "/service-role/"
    description = "Policy used in trust relationship with CodeStar"
    policy      = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Resource": [
        "*"
      ],
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ]
    }
  ]
}
POLICY
}

resource "aws_iam_policy_attachment" "codestar_policy_attachment" {
  name       = "codestar-policy-attachment-%s"
  policy_arn = "${aws_iam_policy.codestar_policy.arn}"
  roles      = ["${aws_iam_role.codestar_role.id}"]
}

resource "aws_codestar_project" "foo" {
  name         = "test-%s"
  description  = "test_codestar_project"
}
`, rName, rName, rName, rName)
}

func testAccAWSCodeStarProjectConfig_basicUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "codestar_role" {
  name = "codestar-role-%s"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codestar.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "codestar_policy" {
    name        = "codestar-policy-%s"
    path        = "/service-role/"
    description = "Policy used in trust relationship with CodeStar"
    policy      = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Resource": [
        "*"
      ],
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ]
    }
  ]
}
POLICY
}

resource "aws_iam_policy_attachment" "codestar_policy_attachment" {
  name       = "codestar-policy-attachment-%s"
  policy_arn = "${aws_iam_policy.codestar_policy.arn}"
  roles      = ["${aws_iam_role.codestar_role.id}"]
}

resource "aws_codestar_project" "foo" {
  name         = "test-%s"
  description  = "pomato"
}
`, rName, rName, rName, rName)
}
