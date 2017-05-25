package aws

import (
	"fmt"
	"strings"
	"testing"
	"unicode"

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
			{
				Config: testAccAWSCodeBuildProjectConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeBuildProjectExists("aws_codebuild_project.foo"),
					resource.TestCheckResourceAttr(
						"aws_codebuild_project.foo", "build_timeout", "5"),
				),
			},
			{
				Config: testAccAWSCodeBuildProjectConfig_basicUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeBuildProjectExists("aws_codebuild_project.foo"),
					resource.TestCheckResourceAttr(
						"aws_codebuild_project.foo", "build_timeout", "50"),
				),
			},
		},
	})
}

func TestAccAWSCodeBuildProject_default_build_timeout(t *testing.T) {
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeBuildProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeBuildProjectConfig_default_timeout(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeBuildProjectExists("aws_codebuild_project.foo"),
					resource.TestCheckResourceAttr(
						"aws_codebuild_project.foo", "build_timeout", "60"),
				),
			},
			{
				Config: testAccAWSCodeBuildProjectConfig_basicUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeBuildProjectExists("aws_codebuild_project.foo"),
					resource.TestCheckResourceAttr(
						"aws_codebuild_project.foo", "build_timeout", "50"),
				),
			},
		},
	})
}

func TestAWSCodeBuildProject_artifactsTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "CODEPIPELINE", ErrCount: 0},
		{Value: "NO_ARTIFACTS", ErrCount: 0},
		{Value: "S3", ErrCount: 0},
		{Value: "XYZ", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildArifactsType(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project artifacts type to trigger a validation error")
		}
	}
}

func TestAWSCodeBuildProject_artifactsNamespaceTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "NONE", ErrCount: 0},
		{Value: "BUILD_ID", ErrCount: 0},
		{Value: "XYZ", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildArifactsNamespaceType(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project artifacts namepsace_type to trigger a validation error")
		}
	}
}

func longTestData() string {
	data := `
	test-test-test-test-test-test-test-test-test-test-
	test-test-test-test-test-test-test-test-test-test-
	test-test-test-test-test-test-test-test-test-test-
	test-test-test-test-test-test-test-test-test-test-
	test-test-test-test-test-test-test-test-test-test-
	test-test-test-test-test-test-test-test-test-test-
	`

	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, data)
}

func TestAWSCodeBuildProject_nameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "_test", ErrCount: 1},
		{Value: "test", ErrCount: 0},
		{Value: "1_test", ErrCount: 0},
		{Value: "test**1", ErrCount: 1},
		{Value: longTestData(), ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildProjectName(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project name to trigger a validation error - %s", errors)
		}
	}
}

func TestAWSCodeBuildProject_descriptionValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "test", ErrCount: 0},
		{Value: longTestData(), ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildProjectDescription(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project description to trigger a validation error")
		}
	}
}

func TestAWSCodeBuildProject_environmentComputeTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "BUILD_GENERAL1_SMALL", ErrCount: 0},
		{Value: "BUILD_GENERAL1_MEDIUM", ErrCount: 0},
		{Value: "BUILD_GENERAL1_LARGE", ErrCount: 0},
		{Value: "BUILD_GENERAL1_VERYLARGE", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildEnvironmentComputeType(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project environment compute_type to trigger a validation error")
		}
	}
}

func TestAWSCodeBuildProject_environmentTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "LINUX_CONTAINER", ErrCount: 0},
		{Value: "WINDOWS_CONTAINER", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildEnvironmentType(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project environment type to trigger a validation error")
		}
	}
}

func TestAWSCodeBuildProject_sourceTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "CODECOMMIT", ErrCount: 0},
		{Value: "CODEPIPELINE", ErrCount: 0},
		{Value: "GITHUB", ErrCount: 0},
		{Value: "S3", ErrCount: 0},
		{Value: "GITLAB", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildSourceType(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project source type to trigger a validation error")
		}
	}
}

func TestAWSCodeBuildProject_sourceAuthTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "OAUTH", ErrCount: 0},
		{Value: "PASSWORD", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildSourceAuthType(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project source auth to trigger a validation error")
		}
	}
}

func TestAWSCodeBuildProject_timeoutValidation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{Value: 10, ErrCount: 0},
		{Value: 200, ErrCount: 0},
		{Value: 1, ErrCount: 1},
		{Value: 500, ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateAwsCodeBuildTimeout(tc.Value, "aws_codebuild_project")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the AWS CodeBuild project timeout to trigger a validation error")
		}
	}
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

func testAccAWSCodeBuildProjectConfig_basic(rName string) string {
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
  build_timeout      = "5"
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
    location = "https://github.com/hashicorp/packer.git"
  }

  tags {
    "Environment" = "Test"
  }
}
`, rName, rName, rName, rName)
}

func testAccAWSCodeBuildProjectConfig_basicUpdated(rName string) string {
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
  build_timeout      = "50"
	service_role = "${aws_iam_role.codebuild_role.arn}"

	artifacts {
		type = "NO_ARTIFACTS"
	}

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "2"
    type         = "LINUX_CONTAINER"

		environment_variable = {
			"name"  = "SOME_OTHERKEY"
			"value" = "SOME_OTHERVALUE"
		}
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/hashicorp/packer.git"
  }

  tags {
    "Environment" = "Test"
  }
}
`, rName, rName, rName, rName)
}

func testAccAWSCodeBuildProjectConfig_default_timeout(rName string) string {
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

	service_role = "${aws_iam_role.codebuild_role.arn}"

	artifacts {
		type = "NO_ARTIFACTS"
	}

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "2"
    type         = "LINUX_CONTAINER"

		environment_variable = {
			"name"  = "SOME_OTHERKEY"
			"value" = "SOME_OTHERVALUE"
		}
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/hashicorp/packer.git"
  }

  tags {
    "Environment" = "Test"
  }
}
`, rName, rName, rName, rName)
}
