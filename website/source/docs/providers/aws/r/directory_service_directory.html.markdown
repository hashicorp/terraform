---
layout: "aws"
page_title: "AWS: aws_directory_service_directory"
sidebar_current: "docs-aws-resource-directory-service-directory"
description: |-
  Provides a directory in AWS Directory Service.
---

# aws\_directory\_service\_directory

Provides a directory in AWS Directory Service.

## Example Usage

```
resource "aws_directory_service_directory" "bar" {
  name = "corp.notexample.com"
  password = "SuperSecretPassw0rd"
  size = "Small"

  vpc_settings {
    vpc_id = "${aws_vpc.main.id}"
    subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.main.id}"
  availability_zone = "us-west-2a"
  cidr_block = "10.0.1.0/24"
}
resource "aws_subnet" "bar" {
  vpc_id = "${aws_vpc.main.id}"
  availability_zone = "us-west-2b"
  cidr_block = "10.0.2.0/24"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The fully qualified name for the directory, such as `corp.example.com`
* `password` - (Required) The password for the directory administrator.
* `size` - (Required) The size of the directory (`Small` or `Large` are accepted values).
* `vpc_settings` - (Required) VPC related information about the directory. Fields documented below.
* `alias` - (Optional) The alias for the directory (must be unique amongst all aliases in AWS). Required for `enable_sso`.
* `description` - (Optional) A textual description for the directory.
* `short_name` - (Optional) The short name of the directory, such as `CORP`.
* `enable_sso` - (Optional) Whether to enable single-sign on for the directory. Requires `alias`. Defaults to `false`.

**vpc\_settings** supports the following:

* `subnet_ids` - (Required) The identifiers of the subnets for the directory servers (min. 2 subnets in 2 different AZs).
* `vpc_id` - (Required) The identifier of the VPC that the directory is in.

## Attributes Reference

The following attributes are exported:

* `id` - The directory identifier.
* `access_url` - The access URL for the directory, such as `http://alias.awsapps.com`.
* `dns_ip_addresses` - A list of IP addresses of the DNS servers for the directory.
* `type` - The directory type.
