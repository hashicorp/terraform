---
layout: "aws"
page_title: "Provider: AWS"
sidebar_current: "docs-aws-index"
---

# AWS Provider

The Amazon Web Services (AWS) provider is used to interact with the
many resources supported by AWS. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the AWS Provider
provider "aws" {
    access_key = "${var.aws_access_key}"
    secret_key = "${var.aws_secret_key}"
    region = "us-east-1"
}

# Create a web server
resource "aws_instance" "web" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `access_key` - (Required) This is the AWS access key. It must be provided, but
  it can also be sourced from the `AWS_ACCESS_KEY` environment variable.

* `secret_key` - (Required) This is the AWS secret key. It must be provided, but
  it can also be sourced from the `AWS_SECRET_KEY` environment variable.

* `region` - (Required) This is the AWS region. It must be provided, but
  it can also be sourced from the `AWS_REGION` environment variables.

