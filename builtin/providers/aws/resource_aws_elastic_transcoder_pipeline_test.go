package aws

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elastictranscoder"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticTranscoderPipeline_basic(t *testing.T) {
	pipeline := &elastictranscoder.Pipeline{}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elastictranscoder_pipeline.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckElasticTranscoderPipelineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: awsElasticTranscoderPipelineConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticTranscoderPipelineExists("aws_elastictranscoder_pipeline.bar", pipeline),
				),
			},
		},
	})
}

func TestAccAWSElasticTranscoderPipeline_notifications(t *testing.T) {
	pipeline := elastictranscoder.Pipeline{}

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elastictranscoder_pipeline.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckElasticTranscoderPipelineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: awsElasticTranscoderNotifications(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticTranscoderPipelineExists("aws_elastictranscoder_pipeline.bar", &pipeline),
					testAccCheckAWSElasticTranscoderPipeline_notifications(&pipeline, []string{"warning", "completed"}),
				),
			},

			// update and check that we have 1 less notification
			resource.TestStep{
				Config: awsElasticTranscoderNotifications_update(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticTranscoderPipelineExists("aws_elastictranscoder_pipeline.bar", &pipeline),
					testAccCheckAWSElasticTranscoderPipeline_notifications(&pipeline, []string{"completed"}),
				),
			},
		},
	})
}

// testAccCheckTags can be used to check the tags on a resource.
func testAccCheckAWSElasticTranscoderPipeline_notifications(
	p *elastictranscoder.Pipeline, notifications []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		var notes []string
		if p.Notifications.Completed != nil && *p.Notifications.Completed != "" {
			notes = append(notes, "completed")
		}
		if p.Notifications.Error != nil && *p.Notifications.Error != "" {
			notes = append(notes, "error")
		}
		if p.Notifications.Progressing != nil && *p.Notifications.Progressing != "" {
			notes = append(notes, "progressing")
		}
		if p.Notifications.Warning != nil && *p.Notifications.Warning != "" {
			notes = append(notes, "warning")
		}

		if len(notes) != len(notifications) {
			return fmt.Errorf("ETC notifications didn't match:\n\texpected: %#v\n\tgot: %#v\n\n", notifications, notes)
		}

		sort.Strings(notes)
		sort.Strings(notifications)

		if !reflect.DeepEqual(notes, notifications) {
			return fmt.Errorf("ETC notifications were not equal:\n\texpected: %#v\n\tgot: %#v\n\n", notifications, notes)
		}

		return nil
	}
}

func TestAccAWSElasticTranscoderPipeline_withContentConfig(t *testing.T) {
	pipeline := &elastictranscoder.Pipeline{}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elastictranscoder_pipeline.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckElasticTranscoderPipelineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: awsElasticTranscoderPipelineWithContentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticTranscoderPipelineExists("aws_elastictranscoder_pipeline.bar", pipeline),
				),
			},
			resource.TestStep{
				Config: awsElasticTranscoderPipelineWithContentConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticTranscoderPipelineExists("aws_elastictranscoder_pipeline.bar", pipeline),
				),
			},
		},
	})
}

func TestAccAWSElasticTranscoderPipeline_withPermissions(t *testing.T) {
	pipeline := &elastictranscoder.Pipeline{}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elastictranscoder_pipeline.baz",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckElasticTranscoderPipelineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: awsElasticTranscoderPipelineWithPerms,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticTranscoderPipelineExists("aws_elastictranscoder_pipeline.baz", pipeline),
				),
			},
		},
	})
}

func testAccCheckAWSElasticTranscoderPipelineExists(n string, res *elastictranscoder.Pipeline) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Pipeline ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elastictranscoderconn

		out, err := conn.ReadPipeline(&elastictranscoder.ReadPipelineInput{
			Id: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		*res = *out.Pipeline

		return nil
	}
}

func testAccCheckElasticTranscoderPipelineDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elastictranscoderconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elastictranscoder_pipline" {
			continue
		}

		out, err := conn.ReadPipeline(&elastictranscoder.ReadPipelineInput{
			Id: aws.String(rs.Primary.ID),
		})

		if err == nil {
			if out.Pipeline != nil && *out.Pipeline.Id == rs.Primary.ID {
				return fmt.Errorf("Elastic Transcoder Pipeline still exists")
			}
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		if awsErr.Code() != "ResourceNotFoundException" {
			return fmt.Errorf("unexpected error: %s", awsErr)
		}

	}
	return nil
}

const awsElasticTranscoderPipelineConfigBasic = `
resource "aws_elastictranscoder_pipeline" "bar" {
  input_bucket  = "${aws_s3_bucket.test_bucket.bucket}"
  output_bucket = "${aws_s3_bucket.test_bucket.bucket}"
  name          = "aws_elastictranscoder_pipeline_tf_test_"
  role          = "${aws_iam_role.test_role.arn}"
}

resource "aws_iam_role" "test_role" {
  name = "aws_elastictranscoder_pipeline_tf_test_role_"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "test_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-test-bucket"
  acl    = "private"
}
`

const awsElasticTranscoderPipelineWithContentConfig = `
resource "aws_elastictranscoder_pipeline" "bar" {
  input_bucket = "${aws_s3_bucket.content_bucket.bucket}"
  name         = "aws_elastictranscoder_pipeline_tf_test_"
  role         = "${aws_iam_role.test_role.arn}"

  content_config {
    bucket        = "${aws_s3_bucket.content_bucket.bucket}"
    storage_class = "Standard"
  }

  thumbnail_config {
    bucket        = "${aws_s3_bucket.content_bucket.bucket}"
    storage_class = "Standard"
  }
}

resource "aws_iam_role" "test_role" {
  name = "aws_elastictranscoder_pipeline_tf_test_role_"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "content_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-content-bucket"
  acl    = "private"
}

resource "aws_s3_bucket" "input_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-input-bucket"
  acl    = "private"
}

resource "aws_s3_bucket" "thumb_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-thumb-bucket"
  acl    = "private"
}
`

const awsElasticTranscoderPipelineWithContentConfigUpdate = `
resource "aws_elastictranscoder_pipeline" "bar" {
  input_bucket = "${aws_s3_bucket.input_bucket.bucket}"
  name         = "aws_elastictranscoder_pipeline_tf_test_"
  role         = "${aws_iam_role.test_role.arn}"

  content_config {
    bucket        = "${aws_s3_bucket.content_bucket.bucket}"
    storage_class = "Standard"
  }

  thumbnail_config {
    bucket        = "${aws_s3_bucket.thumb_bucket.bucket}"
    storage_class = "Standard"
  }
}

resource "aws_iam_role" "test_role" {
  name = "aws_elastictranscoder_pipeline_tf_test_role_"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "content_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-content-bucket"
  acl    = "private"
}

resource "aws_s3_bucket" "input_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-input-bucket"
  acl    = "private"
}

resource "aws_s3_bucket" "thumb_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-thumb-bucket"
  acl    = "private"
}
`

const awsElasticTranscoderPipelineWithPerms = `
resource "aws_elastictranscoder_pipeline" "baz" {
  input_bucket = "${aws_s3_bucket.content_bucket.bucket}"
  name         = "aws_elastictranscoder_pipeline_tf_test_"
  role         = "${aws_iam_role.test_role.arn}"

  content_config {
    bucket        = "${aws_s3_bucket.content_bucket.bucket}"
    storage_class = "Standard"
  }

  content_config_permissions = {
    grantee_type = "Group"
    grantee      = "AuthenticatedUsers"
    access       = ["FullControl"]
  }

  thumbnail_config {
    bucket        = "${aws_s3_bucket.content_bucket.bucket}"
    storage_class = "Standard"
  }

  thumbnail_config_permissions = {
    grantee_type = "Group"
    grantee      = "AuthenticatedUsers"
    access       = ["FullControl"]
  }
}

resource "aws_iam_role" "test_role" {
  name = "aws_elastictranscoder_pipeline_tf_test_role_"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "content_bucket" {
  bucket = "aws-elasticencoder-pipeline-tf-content-bucket"
  acl    = "private"
}
`

func awsElasticTranscoderNotifications(r int) string {
	return fmt.Sprintf(`
resource "aws_elastictranscoder_pipeline" "bar" {
  input_bucket  = "${aws_s3_bucket.test_bucket.bucket}"
  output_bucket = "${aws_s3_bucket.test_bucket.bucket}"
  name          = "tf-transcoder-%d"
  role          = "${aws_iam_role.test_role.arn}"

  notifications {
    completed = "${aws_sns_topic.topic_example.arn}"
    warning   = "${aws_sns_topic.topic_example.arn}"
  }
}

resource "aws_iam_role" "test_role" {
  name = "tf-transcoder-%d"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "test_bucket" {
  bucket = "tf-transcoder-%d"
  acl    = "private"
}

resource "aws_sns_topic" "topic_example" {
  name = "tf-transcoder-%d"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Id": "AWSAccountTopicAccess",
  "Statement": [
    {
      "Sid": "*",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sns:Publish",
      "Resource": "*"
    }
  ]
}
EOF
}`, r, r, r, r)
}

func awsElasticTranscoderNotifications_update(r int) string {
	return fmt.Sprintf(`
resource "aws_elastictranscoder_pipeline" "bar" {
  input_bucket  = "${aws_s3_bucket.test_bucket.bucket}"
  output_bucket = "${aws_s3_bucket.test_bucket.bucket}"
  name          = "tf-transcoder-%d"
  role          = "${aws_iam_role.test_role.arn}"

  notifications {
    completed = "${aws_sns_topic.topic_example.arn}"
  }
}

resource "aws_iam_role" "test_role" {
  name = "tf-transcoder-%d"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "test_bucket" {
  bucket = "tf-transcoder-%d"
  acl    = "private"
}

resource "aws_sns_topic" "topic_example" {
  name = "tf-transcoder-%d"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Id": "AWSAccountTopicAccess",
  "Statement": [
    {
      "Sid": "*",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sns:Publish",
      "Resource": "*"
    }
  ]
}
EOF
}`, r, r, r, r)
}
