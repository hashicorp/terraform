---
layout: "aws"
page_title: "AWS: aws_elastic_beanstalk_environment"
sidebar_current: "docs-aws-resource-elastic-beanstalk-environment"
description: |-
  Provides an Elastic Beanstalk Environment Resource
---

# aws\_elastic\_beanstalk\_<wbr>Environment

Provides an Elastic Beanstalk Environment Resource. Elastic Beanstalk allows 
you to deploy and manage applications in the AWS cloud without worrying about 
the infrastructure that runs those applications.

This resource creates an application that has one configuration template named 
`default`, and no application versions

## Example Usage

Stacks: http://docs.aws.amazon.com/fr_fr/elasticbeanstalk/latest/dg/concepts.platforms.html

```
resource "aws_eip" "lb" {
    instance = "${aws_instance.web.id}"
    vpc = true
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

