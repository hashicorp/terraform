package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeBuildProject_basic(t *testing.T) {
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeBuildProjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeBuildProjectConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeBuildProjectExists("aws_codebuild_project.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSCodeBuildProjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No CodeBuild Project ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).codebuildconn

		out, err := conn.BatchGetProjects(&codebuild.BatchGetProjectsInput{
			Names: []*string{
				aws.String(rs.Primary.ID),
			},
		})

		if err != nil {
			return err
		}

		if len(out.Projects) < 1 {
			return fmt.Errorf("No project found")
		}

		return nil
	}
}

func testAccCheckAWSCodeBuildProjectDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codebuildconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codebuild_project" {
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

func testAccAWSCodeBuildProjectConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "codebuild_role" {
  name = "codebuild-role-%s"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codebuild.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "codebuild_policy" {
    name        = "codebuild-policy-%s"
    path        = "/service-role/"
    description = "Policy used in trust relationship with CodeBuild"
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

resource "aws_iam_policy_attachment" "codebuild_policy_attachment" {
  name       = "codebuild-policy-attachment-%s"
  policy_arn = "${aws_iam_policy.codebuild_policy.arn}"
  roles      = ["${aws_iam_role.codebuild_role.id}"]
}

resource "aws_codebuild_project" "foo" {
  name         = "test-project-%s"
  description  = "test_codebuild_project"
  timeout      = "5"
	service_role = "${aws_iam_role.codebuild_role.arn}"

	artifacts {
		type = "NO_ARTIFACTS"
	}

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "2"
    type         = "LINUX_CONTAINER"
		
		environment_variable = {
			"name"  = "SOME_KEY"
			"value" = "SOME_VALUE"
		}
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/mitchellh/packer.git"
  }

  tags {
    "Environment" = "Test"
  }
}
`, rName, rName, rName, rName)
}
