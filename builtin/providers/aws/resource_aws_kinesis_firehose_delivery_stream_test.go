package aws

import (
	"fmt"
	"log"
	"math/rand"
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
	config := fmt.Sprintf(testAccKinesisFirehoseDeliveryStreamConfig_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
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
	preconfig := fmt.Sprintf(testAccKinesisFirehoseDeliveryStreamConfig_s3, ri, ri)
	postConfig := fmt.Sprintf(testAccKinesisFirehoseDeliveryStreamConfig_s3Updates, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
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
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	name = "terraform-kinesis-firehose-basictest-%d"
	destination = "s3"
	role_arn = "arn:aws:iam::946579370547:role/firehose_delivery_role"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
}`

var testAccKinesisFirehoseDeliveryStreamConfig_s3 = `
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	name = "terraform-kinesis-firehose-s3test-%d"
	destination = "s3"
	role_arn = "arn:aws:iam::946579370547:role/firehose_delivery_role"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
}`

var testAccKinesisFirehoseDeliveryStreamConfig_s3Updates = `
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-01-%d"
	acl = "private"
}

resource "aws_kinesis_firehose_delivery_stream" "test_stream" {
	name = "terraform-kinesis-firehose-s3test-%d"
	destination = "s3"
	role_arn = "arn:aws:iam::946579370547:role/firehose_delivery_role"
	s3_bucket_arn = "${aws_s3_bucket.bucket.arn}"
	s3_buffer_size = 10
	s3_buffer_interval = 400
	s3_data_compression = "GZIP"
}`
