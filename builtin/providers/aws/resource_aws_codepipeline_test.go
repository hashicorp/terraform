package aws

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodePipeline_basic(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Environment variable GITHUB_TOKEN is not set")
	}

	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodePipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodePipelineConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodePipelineExists("aws_codepipeline.bar"),
					resource.TestCheckResourceAttr("aws_codepipeline.bar", "artifact_store.0.type", "S3"),
					resource.TestCheckResourceAttr("aws_codepipeline.bar", "artifact_store.0.encryption_key.0.id", "1234"),
					resource.TestCheckResourceAttr("aws_codepipeline.bar", "artifact_store.0.encryption_key.0.type", "KMS"),
				),
			},
			{
				Config: testAccAWSCodePipelineConfig_basicUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodePipelineExists("aws_codepipeline.bar"),
					resource.TestCheckResourceAttr("aws_codepipeline.bar", "artifact_store.0.type", "S3"),
					resource.TestCheckResourceAttr("aws_codepipeline.bar", "artifact_store.0.encryption_key.0.id", "4567"),
					resource.TestCheckResourceAttr("aws_codepipeline.bar", "artifact_store.0.encryption_key.0.type", "KMS"),
				),
			},
		},
	})
}

func testAccCheckAWSCodePipelineExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No CodePipeline ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).codepipelineconn

		_, err := conn.GetPipeline(&codepipeline.GetPipelineInput{
			Name: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAWSCodePipelineDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codepipelineconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codepipeline" {
			continue
		}

		_, err := conn.GetPipeline(&codepipeline.GetPipelineInput{
			Name: aws.String(rs.Primary.ID),
		})

		if err == nil {
			return fmt.Errorf("Expected AWS CodePipeline to be gone, but was still found")
		}
		return nil
	}

	return fmt.Errorf("Default error in CodePipeline Test")
}

func testAccAWSCodePipelineConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "foo" {
  bucket = "tf-test-pipeline-%s"
  acl    = "private"
}

resource "aws_iam_role" "codepipeline_role" {
  name = "codepipeline-role-%s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codepipeline.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "codepipeline_policy" {
  name = "codepipeline_policy"
  role = "${aws_iam_role.codepipeline_role.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect":"Allow",
      "Action": [
        "s3:GetObject",
        "s3:GetObjectVersion",
        "s3:GetBucketVersioning"
      ],
      "Resource": [
        "${aws_s3_bucket.foo.arn}",
        "${aws_s3_bucket.foo.arn}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "codebuild:BatchGetBuilds",
        "codebuild:StartBuild"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_codepipeline" "bar" {
  name     = "test-pipeline-%s"
  role_arn = "${aws_iam_role.codepipeline_role.arn}"

  artifact_store {
    location = "${aws_s3_bucket.foo.bucket}"
    type     = "S3"

    encryption_key {
      id   = "1234"
      type = "KMS"
    }
  }

  stage {
    name = "Source"

    action {
      name             = "Source"
      category         = "Source"
      owner            = "ThirdParty"
      provider         = "GitHub"
      version          = "1"
      output_artifacts = ["test"]

      configuration {
        Owner  = "lifesum-terraform"
        Repo   = "test"
        Branch = "master"
      }
    }
  }

  stage {
    name = "Build"

    action {
      name            = "Build"
      category        = "Build"
      owner           = "AWS"
      provider        = "CodeBuild"
      input_artifacts = ["test"]
      version         = "1"

      configuration {
        ProjectName = "test"
      }
    }
  }
}
`, rName, rName, rName)
}

func testAccAWSCodePipelineConfig_basicUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "foo" {
  bucket = "tf-test-pipeline-%s"
  acl    = "private"
}

resource "aws_iam_role" "codepipeline_role" {
  name = "codepipeline-role-%s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codepipeline.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "codepipeline_policy" {
  name = "codepipeline_policy"
  role = "${aws_iam_role.codepipeline_role.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect":"Allow",
      "Action": [
        "s3:GetObject",
        "s3:GetObjectVersion",
        "s3:GetBucketVersioning"
      ],
      "Resource": [
        "${aws_s3_bucket.foo.arn}",
        "${aws_s3_bucket.foo.arn}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "codebuild:BatchGetBuilds",
        "codebuild:StartBuild"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_codepipeline" "bar" {
  name     = "test-pipeline-%s"
  role_arn = "${aws_iam_role.codepipeline_role.arn}"

  artifact_store {
    location = "${aws_s3_bucket.foo.bucket}"
    type     = "S3"

    encryption_key {
      id   = "4567"
      type = "KMS"
    }
  }

  stage {
    name = "Source"

    action {
      name             = "Source"
      category         = "Source"
      owner            = "ThirdParty"
      provider         = "GitHub"
      version          = "1"
      output_artifacts = ["bar"]

      configuration {
        Owner  = "foo-terraform"
        Repo   = "bar"
        Branch = "stable"
      }
    }
  }

  stage {
    name = "Build"

    action {
      name            = "Build"
      category        = "Build"
      owner           = "AWS"
      provider        = "CodeBuild"
      input_artifacts = ["bar"]
      version         = "1"

      configuration {
        ProjectName = "foo"
      }
    }
  }
}
`, rName, rName, rName)
}
