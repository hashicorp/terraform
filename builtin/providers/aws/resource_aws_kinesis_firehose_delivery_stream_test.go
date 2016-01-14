package aws

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKinesisFirehoseDeliveryStream_basic(t *testing.T) {
	var stream firehose.DeliveryStreamDescription
	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	config := fmt.Sprintf(testAccKinesisFirehoseDeliveryStreamConfig_basic,
		os.Getenv("AWS_ACCOUNT_ID"), ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("AWS_ACCOUNT_ID") == "" {
				t.Fatal("AWS_ACCOUNT_ID must be set")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKinesisFirehoseDeliveryStreamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKinesisFirehoseDeliveryStreamExists("aws_kinesis_firehose_delivery_stream.test_stream", &stream),
					testAccCheckAWSKinesisFirehoseDeliveryStreamAttributes(&stream),
				),
			},
		},
	})
}

func TestAccAWSKinesisFirehoseDeliveryStream_s3ConfigUpdates(t *testing.T) {
	var stream firehose.DeliveryStreamDescription

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	preconfig := fmt.Sprintf(testAccKinesisFirehoseDeliveryStreamConfig_s3,
		os.Getenv("AWS_ACCOUNT_ID"), ri, ri)
	postConfig := fmt.Sprintf(testAccKinesisFirehoseDeliveryStreamConfig_s3Updates,
		os.Getenv("AWS_ACCOUNT_ID"), ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("AWS_ACCOUNT_ID") == "" {
				t.Fatal("AWS_ACCOUNT_ID must be set")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKinesisFirehoseDeliveryStreamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preconfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKinesisFirehoseDeliveryStreamExists("aws_kinesis_firehose_delivery_stream.test_stream", &stream),
					testAccCheckAWSKinesisFirehoseDeliveryStreamAttributes(&stream),
					resource.TestCheckResourceAttr(
						"aws_kinesis_firehose_delivery_stream.test_stream", "s3_buffer_size", "5"),
					resource.TestCheckResourceAttr(
						"aws_kinesis_firehose_delivery_stream.test_stream", "s3_buffer_interval", "300"),
					resource.TestCheckResourceAttr(
						"aws_kinesis_firehose_delivery_stream.test_stream", "s3_data_compression", "UNCOMPRESSED"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKinesisFirehoseDeliveryStreamExists("aws_kinesis_firehose_delivery_stream.test_stream", &stream),
					testAccCheckAWSKinesisFirehoseDeliveryStreamAttributes(&stream),
					resource.TestCheckResourceAttr(
						"aws_kinesis_firehose_delivery_stream.test_stream", "s3_buffer_size", "10"),
					resource.TestCheckResourceAttr(
						"aws_kinesis_firehose_delivery_stream.test_stream", "s3_buffer_interval", "400"),
					resource.TestCheckResourceAttr(
						"aws_kinesis_firehose_delivery_stream.test_stream", "s3_data_compression", "GZIP"),
				),
			},
		},
	})
}

func testAccCheckKinesisFirehoseDeliveryStreamExists(n string, stream *firehose.DeliveryStreamDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		log.Printf("State: %#v", s.RootModule().Resources)
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Kinesis Firehose ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).firehoseconn
		describeOpts := &firehose.DescribeDeliveryStreamInput{
			DeliveryStreamName: aws.String(rs.Primary.Attributes["name"]),
		}
		resp, err := conn.DescribeDeliveryStream(describeOpts)
		if err != nil {
			return err
		}

		*stream = *resp.DeliveryStreamDescription

		return nil
	}
}

func testAccCheckAWSKinesisFirehoseDeliveryStreamAttributes(stream *firehose.DeliveryStreamDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.HasPrefix(*stream.DeliveryStreamName, "terraform-kinesis-firehose") {
			return fmt.Errorf("Bad Stream name: %s", *stream.DeliveryStreamName)
		}
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_kinesis_firehose_delivery_stream" {
				continue
			}
			if *stream.DeliveryStreamARN != rs.Primary.Attributes["arn"] {
				return fmt.Errorf("Bad Delivery Stream ARN\n\t expected: %s\n\tgot: %s\n", rs.Primary.Attributes["arn"], *stream.DeliveryStreamARN)
			}
		}
		return nil
	}
}

func testAccCheckKinesisFirehoseDeliveryStreamDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_kinesis_firehose_delivery_stream" {
			continue
		}
		conn := testAccProvider.Meta().(*AWSClient).firehoseconn
		describeOpts := &firehose.DescribeDeliveryStreamInput{
			DeliveryStreamName: aws.String(rs.Primary.Attributes["name"]),
		}
		resp, err := conn.DescribeDeliveryStream(describeOpts)
		if err == nil {
			if resp.DeliveryStreamDescription != nil && *resp.DeliveryStreamDescription.DeliveryStreamStatus != "DELETING" {
				return fmt.Errorf("Error: Delivery Stream still exists")
			}
		}

		return nil

	}

	return nil
}

var testAccKinesisFirehoseDeliveryStreamConfig_basic = `
resource "aws_iam_role" "firehose" {
	name = "terraform_acctest_firehose_delivery_role"
	assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "firehose.amazonaws.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "%s"
        }
      }
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
}

resource "aws_iam_role_policy" "firehose" {
	name = "terraform_acctest_firehose_delivery_policy"
	role = "${aws_iam_role.firehose.id}"
	policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:GetBucketLocation",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads",
        "s3:PutObject"
      ],
      "Resource": [
        "arn:aws:s3:::${aws_s3_bucket.bucket.id}",
        "arn:aws:s3:::${aws_s3_bucket.bucket.id}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	depends_on = ["aws_iam_role_policy.firehose"]
	name = "terraform-kinesis-firehose-basictest-%d"
	destination = "s3"
	role_arn = "${aws_iam_role.firehose.arn}"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
}`

var testAccKinesisFirehoseDeliveryStreamConfig_s3 = `
resource "aws_iam_role" "firehose" {
	name = "terraform_acctest_firehose_delivery_role_s3"
	assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "firehose.amazonaws.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "%s"
        }
      }
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
}

resource "aws_iam_role_policy" "firehose" {
	name = "terraform_acctest_firehose_delivery_policy_s3"
	role = "${aws_iam_role.firehose.id}"
	policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:GetBucketLocation",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads",
        "s3:PutObject"
      ],
      "Resource": [
        "arn:aws:s3:::${aws_s3_bucket.bucket.id}",
        "arn:aws:s3:::${aws_s3_bucket.bucket.id}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	depends_on = ["aws_iam_role_policy.firehose"]
	name = "terraform-kinesis-firehose-s3test-%d"
	destination = "s3"
	role_arn = "${aws_iam_role.firehose.arn}"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
}`

var testAccKinesisFirehoseDeliveryStreamConfig_s3Updates = `
resource "aws_iam_role" "firehose" {
	name = "terraform_acctest_firehose_delivery_role_s3"
	assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "firehose.amazonaws.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "%s"
        }
      }
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
}

resource "aws_iam_role_policy" "firehose" {
	name = "terraform_acctest_firehose_delivery_policy_s3"
	role = "${aws_iam_role.firehose.id}"
	policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:GetBucketLocation",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads",
        "s3:PutObject"
      ],
      "Resource": [
        "arn:aws:s3:::${aws_s3_bucket.bucket.id}",
        "arn:aws:s3:::${aws_s3_bucket.bucket.id}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	depends_on = ["aws_iam_role_policy.firehose"]
	name = "terraform-kinesis-firehose-s3test-%d"
	destination = "s3"
	role_arn = "${aws_iam_role.firehose.arn}"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
	s3_buffer_size = 10
	s3_buffer_interval = 400
	s3_data_compression = "GZIP"
}`
