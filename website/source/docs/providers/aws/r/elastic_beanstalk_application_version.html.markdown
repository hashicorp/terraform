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

~> **NOTE on Application Version Resource:**  When using the Application Version resource with multiple 
[Elastic Beanstalk Environments](elastic_beanstalk_environment.html) it is possible that an error may be returned
when attempting to delete an Application Version while it is still in use by a different environment.
To work around this you can:
<ol>
<li>Create each environment in a separate AWS account</li>
<li>Create your `aws_elastic_beanstalk_application_version` resources with a unique names in your 
Elastic Beanstalk Application. For example &lt;revision&gt;-&lt;environment&gt;.</li>
</ol>

## Example Usage

```hcl
resource "aws_s3_bucket" "default" {
  bucket = "tftest.applicationversion.bucket"
}

resource "aws_s3_bucket_object" "default" {
  bucket = "${aws_s3_bucket.default.id}"
  key    = "beanstalk/go-v1.zip"
  source = "go-v1.zip"
}

resource "aws_elastic_beanstalk_application" "default" {
  name        = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_application_version" "default" {
  name        = "tf-test-version-label"
  application = "tf-test-name"
  description = "application version created by terraform"
  bucket      = "${aws_s3_bucket.default.id}"
  key         = "${aws_s3_bucket_object.default.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the this Application Version.
* `application` - (Required) Name of the Beanstalk Application the version is associated with.
* `description` - (Optional) Short description of the Application Version.
* `bucket` - (Required) S3 bucket that contains the Application Version source bundle.
* `key` - (Required) S3 object that is the Application Version source bundle.
* `force_delete` - (Optional) On delete, force an Application Version to be deleted when it may be in use
  by multiple Elastic Beanstalk Environments.

## Attributes Reference

The following attributes are exported:

* `name` - The Application Version name.
