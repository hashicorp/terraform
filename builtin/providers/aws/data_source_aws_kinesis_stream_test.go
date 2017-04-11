package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSKinesisStreamDataSource(t *testing.T) {
	var stream kinesis.StreamDescription

	sn := fmt.Sprintf("terraform-kinesis-test-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccCheckAwsKinesisStreamDataSourceConfig, sn)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKinesisStreamDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKinesisStreamExists("aws_kinesis_stream.test_stream", &stream),
					resource.TestCheckResourceAttrSet("data.aws_kinesis_stream.test_stream", "arn"),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "name", sn),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "open_shards.#", "2"),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "closed_shards.#", "0"),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "shard_level_metrics.#", "2"),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "retention_period", "72"),
					resource.TestCheckResourceAttrSet("data.aws_kinesis_stream.test_stream", "creation_timestamp"),
					resource.TestCheckResourceAttr("data.aws_kinesis_stream.test_stream", "tags.Name", "tf-test"),
				),
			},
		},
	})
}

var testAccCheckAwsKinesisStreamDataSourceConfig = `
resource "aws_kinesis_stream" "test_stream" {
	name = "%s"
	shard_count = 2
	retention_period = 72
	tags {
		Name = "tf-test"
	}
	shard_level_metrics = [
		"IncomingBytes",
		"OutgoingBytes"
	]
}

data "aws_kinesis_stream" "test_stream" {
	name = "${aws_kinesis_stream.test_stream.name}"
}
`
