---
layout: "aws"
page_title: "AWS: aws_elastic_beanstalk_application_version"
sidebar_current: "docs-aws-resource-elastic-beanstalk-application-version"
description: |-
  Provides an Elastic Beanstalk Application Version Resource
---

# aws\_elastic\_beanstalk\_application\_<wbr>version

Provides an Elastic Beanstalk Application Version Resource. Elastic Beanstalk allows 
you to deploy and manage applications in the AWS cloud without worrying about 
the infrastructure that runs those applications.

This resource creates a Beanstalk Application Version that can be deployed to a Beanstalk 
Environment.

## Example Usage

```
resource "aws_s3_bucket" "default" {
  bucket = "tftest.applicationversion.bucket"
}

resource "aws_s3_bucket_object" "default" {
  bucket = "${aws_s3_bucket.default.id}"
  key = "beanstalk/go-v1.zip"
  source = "go-v1.zip"
}

resource "aws_elastic_beanstalk_application" "default" {
  name = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_application_version" "default" {
  name = "tf-test-version-label"
  application = "tf-test-name"
  description = "application version created by terraform"
  bucket = "${aws_s3_bucket.default.id}"
  key = "${aws_s3_bucket_object.default.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the this Application Version.
* `application` – (Required) Name of the application that contains the version 
  to be deployed.
* `description` - (Optional) Short description of the Application Version.
* `bucket` - (Required) Name of S3 bucket where the Application Version is stored.
* `key` - (Required) Name of S3 object that contains the Application Version.

## Attributes Reference

The following attributes are exported:

* `name` - The Application Version name.



