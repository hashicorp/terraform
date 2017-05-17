---
layout: "aws"
page_title: "AWS: aws_dms_endpoint"
sidebar_current: "docs-aws-resource-dms-endpoint"
description: |-
  Provides a DMS (Data Migration Service) endpoint resource.
---

# aws\_dms\_endpoint

Provides a DMS (Data Migration Service) endpoint resource. DMS endpoints can be created, updated, deleted, and imported.

~> **Note:** All arguments including the password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
# Create a new endpoint
resource "aws_dms_endpoint" "test" {
  certificate_arn             = "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"
  database_name               = "test"
  endpoint_id                 = "test-dms-endpoint-tf"
  endpoint_type               = "source"
  engine_name                 = "aurora"
  extra_connection_attributes = ""
  kms_key_arn                 = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  password                    = "test"
  port                        = 3306
  server_name                 = "test"
  ssl_mode                    = "none"

  tags {
    Name = "test"
  }

  username = "test"
}
```

## Argument Reference

The following arguments are supported:

* `certificate_arn` - (Optional, Default: empty string) The Amazon Resource Name (ARN) for the certificate.
* `database_name` - (Optional) The name of the endpoint database.
* `endpoint_id` - (Required) The database endpoint identifier.

    - Must contain from 1 to 255 alphanumeric characters or hyphens.
    - Must begin with a letter
    - Must contain only ASCII letters, digits, and hyphens
    - Must not end with a hyphen
    - Must not contain two consecutive hyphens

* `endpoint_type` - (Required) The type of endpoint. Can be one of `source | target`.
* `engine_name` - (Required) The type of engine for the endpoint. Can be one of `mysql | oracle | postgres | mariadb | aurora | redshift | sybase | sqlserver`.
* `extra_connection_attributes` - (Optional) Additional attributes associated with the connection. For available attributes see [Using Extra Connection Attributes with AWS Database Migration Service](http://docs.aws.amazon.com/dms/latest/userguide/CHAP_Introduction.ConnectionAttributes.html).
* `kms_key_arn` - (Optional) The Amazon Resource Name (ARN) for the KMS key that will be used to encrypt the connection parameters. If you do not specify a value for `kms_key_arn`, then AWS DMS will use your default encryption key. AWS KMS creates the default encryption key for your AWS account. Your AWS account has a different default encryption key for each AWS region.
* `password` - (Required) The password to be used to login to the endpoint database.
* `port` - (Required) The port used by the endpoint database.
* `server_name` - (Required) The host name of the server.
* `ssl_mode` - (Optional, Default: none) The SSL mode to use for the connection. Can be one of `none | require | verify-ca | verify-full`
* `tags` - (Optional) A mapping of tags to assign to the resource.
* `username` - (Required) The user name to be used to login to the endpoint database.

## Attributes Reference

The following attributes are exported:

* `endpoint_arn` - The Amazon Resource Name (ARN) for the endpoint.

## Import

Endpoints can be imported using the `endpoint_id`, e.g.

```
$ terraform import aws_dms_endpoint.test test-dms-endpoint-tf
```
