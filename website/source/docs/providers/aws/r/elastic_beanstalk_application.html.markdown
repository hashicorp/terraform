---
layout: "aws"
page_title: "AWS: aws_elastic_beanstalk_application"
sidebar_current: "docs-aws-resource-elastic-beanstalk-application"
description: |-
  Provides an Elastic Beanstalk Application Resource
---

# aws\_elastic\_beanstalk\_<wbr>application

Provides an Elastic Beanstalk Application Resource. Elastic Beanstalk allows
you to deploy and manage applications in the AWS cloud without worrying about
the infrastructure that runs those applications.

This resource creates an application that has one configuration template named
`default`, and no application versions

## Example Usage

```hcl
resource "aws_elastic_beanstalk_application" "tftest" {
  name        = "tf-test-name"
  description = "tf-test-desc"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application, must be unique within your account
* `description` - (Optional) Short description of the application

## Attributes Reference

The following attributes are exported:

* `name`
* `description`


## Import

Elastic Beanstalk Applications can be imported using the `name`, e.g.

```
$ terraform import aws_elastic_beanstalk_application.tf_test tf-test-name
```