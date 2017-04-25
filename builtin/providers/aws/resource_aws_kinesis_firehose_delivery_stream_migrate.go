package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsKinesisFirehoseMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Kinesis Firehose Delivery Stream State v0; migrating to v1")
		return migrateKinesisFirehoseV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateKinesisFirehoseV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty Kinesis Firehose Delivery State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	// migrate flate S3 configuration to a s3_configuration block
	// grab initial values
	is.Attributes["s3_configuration.#"] = "1"
	// Required parameters
	is.Attributes["s3_configuration.0.role_arn"] = is.Attributes["role_arn"]
	is.Attributes["s3_configuration.0.bucket_arn"] = is.Attributes["s3_bucket_arn"]

	// Optional parameters
	if is.Attributes["s3_buffer_size"] != "" {
		is.Attributes["s3_configuration.0.buffer_size"] = is.Attributes["s3_buffer_size"]
	}
	if is.Attributes["s3_data_compression"] != "" {
		is.Attributes["s3_configuration.0.compression_format"] = is.Attributes["s3_data_compression"]
	}
	if is.Attributes["s3_buffer_interval"] != "" {
		is.Attributes["s3_configuration.0.buffer_interval"] = is.Attributes["s3_buffer_interval"]
	}
	if is.Attributes["s3_prefix"] != "" {
		is.Attributes["s3_configuration.0.prefix"] = is.Attributes["s3_prefix"]
	}

	delete(is.Attributes, "role_arn")
	delete(is.Attributes, "s3_bucket_arn")
	delete(is.Attributes, "s3_buffer_size")
	delete(is.Attributes, "s3_data_compression")
	delete(is.Attributes, "s3_buffer_interval")
	delete(is.Attributes, "s3_prefix")

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
