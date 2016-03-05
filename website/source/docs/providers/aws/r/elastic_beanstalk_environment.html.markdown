---
layout: "aws"
page_title: "AWS: aws_elastic_beanstalk_environment"
sidebar_current: "docs-aws-resource-elastic-beanstalk-environment"
description: |-
  Provides an Elastic Beanstalk Environment Resource
---

# aws\_elastic\_beanstalk\_<wbr>environment

Provides an Elastic Beanstalk Environment Resource. Elastic Beanstalk allows 
you to deploy and manage applications in the AWS cloud without worrying about 
the infrastructure that runs those applications.

Environments are often things such as `development`, `integration`, or 
`production`.

## Example Usage


```
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name = "tf-test-name"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux 2015.03 v2.0.3 running Go 1.4"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the this Environment. This name is used 
  in the application URL
* `application` – (Required) Name of the application that contains the version 
  to be deployed
* `description` - (Optional) Short description of the Environment 
* `setting` – (Optional) Option settings to configure the new Environment. These
  override specific values that are set as defaults. The format is detailed
  below in [Option Settings](#option-settings)
* `solution_stack_name` – (Optional) A solution stack to base your environment
off of. Example stacks can be found in the [Amazon API documentation][1]
* `template_name` – (Optional) The name of the Elastic Beanstalk Configuration 
  template to use in deployment
* `tags` – (Optional) A set of tags to apply to the Environment. **Note:** at
this time the Elastic Beanstalk API does not provide a programatic way of
changing these tags after initial application


<a id="option-settings"></a>
## Option Settings

The `setting` and `all_settings` mappings support the following format:

* `namespace` - (Optional) unique namespace identifying the option's 
  associated AWS resource
* `name` - (Optional) name of the configuration option
* `value` - (Optional) value for the configuration option

## Attributes Reference

The following attributes are exported:

* `name`
* `description`
* `application` – the application specified
* `setting` – Settings specifically set for this Environment
* `all_settings` – List of all option settings configured in the Environment. These
  are a combination of default settings and their overrides from `settings` in
  the configuration 


[1]: http://docs.aws.amazon.com/fr_fr/elasticbeanstalk/latest/dg/concepts.platforms.html


