---
layout: "aws"
page_title: "AWS: aws_emr_security_configuraiton"
sidebar_current: "docs-aws-resource-emr-security-configuration"
description: |-
  Provides a resource to manage AWS EMR Security Configurations
---

# aws\_emr\_security\_configuration

Provides a resource to manage AWS EMR Security Configurations

## Example Usage

```hcl
resource "aws_emr_security_configuration" "foo" {
  name = "emrsc_other"

  configuration = <<EOF
{
  "EncryptionConfiguration": {
    "AtRestEncryptionConfiguration": {
      "S3EncryptionConfiguration": {
        "EncryptionMode": "SSE-S3"
      },
      "LocalDiskEncryptionConfiguration": {
        "EncryptionKeyProviderType": "AwsKms",
        "AwsKmsKey": "arn:aws:kms:us-west-2:187416307283:alias/tf_emr_test_key"
      }
    },
    "EnableInTransitEncryption": false,
    "EnableAtRestEncryption": true
  }
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) A unique name for this Security Configuration
* `name_prefix` - (Optional) A prefix for the name of this Security Configuration. 
  Terraform will generate a unique suffix.E.g: `tf-emr-sc-1234`
* `configuration` - (Required) A JSON formatted Security Configuration

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the EMR Security Configuration (Same as the `name`)
* `name` - The Name of the EMR Security Configuration
* `configuration` - The JSON formatted Security Configuration
* `creation_date` - Date the Security Configuration was created

## Import

EMR Security Configurations can be imported using the `name`, e.g.

```
$ terraform import aws_emr_security_configuraiton.sc example-sc-name
```
