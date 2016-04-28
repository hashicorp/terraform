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

* `name` - (Required) A unique name for this Environment. This name is used
  in the application URL
* `application` – (Required) Name of the application that contains the version
  to be deployed
* `cname_prefix` - (Optional) Prefix to use for the fully qualified DNS name of
  the Environment.
* `description` - (Optional) Short description of the Environment
* `tier` - (Optional) Elastic Beanstalk Environment tier. Valid values are `Worker`
  or `WebServer`. If tier is left blank `WebServer` will be used.
* `setting` – (Optional) Option settings to configure the new Environment. These
  override specific values that are set as defaults. The format is detailed
  below in [Option Settings](#option-settings)
* `solution_stack_name` – (Optional) A solution stack to base your environment
off of. Example stacks can be found in the [Amazon API documentation][1]
* `template_name` – (Optional) The name of the Elastic Beanstalk Configuration
  template to use in deployment
* `wait_for_ready_timeout` - (Default: "10m") The maximum
  [duration](https://golang.org/pkg/time/#ParseDuration) that Terraform should
  wait for an Elastic Beanstalk Environment to be in a ready state before timing
  out.
* `tags` – (Optional) A set of tags to apply to the Environment. **Note:** at
this time the Elastic Beanstalk API does not provide a programatic way of
changing these tags after initial application


## Option Settings

Some options can be stack-specific, check [AWS Docs](http://docs.aws.amazon.com/elasticbeanstalk/latest/dg/command-options-general.html)
for supported options and examples.

The `setting` and `all_settings` mappings support the following format:

* `namespace` - (Optional) unique namespace identifying the option's
  associated AWS resource
* `name` - (Optional) name of the configuration option
* `value` - (Optional) value for the configuration option

## Attributes Reference

The following attributes are exported:

* `name` - Name of the Elastic Beanstalk Environment.
* `description` - Description of the Elastic Beanstalk Environment.
* `tier` - The environment tier specified.
* `application` – The Elastic Beanstalk Application specified for this environment.
* `setting` – Settings specifically set for this Environment.
* `all_settings` – List of all option settings configured in the Environment. These
  are a combination of default settings and their overrides from `setting` in
  the configuration.
* `cname` - Fully qualified DNS name for the Environment.
* `autoscaling_groups` - The autoscaling groups used by this environment.
* `instances` - Instances used by this environment.
* `launch_configurations` - Launch configurations in use by this environment.
* `load_balancers` - Elastic load balancers in use by this environment.
* `queues` - SQS queues in use by this environment.
* `triggers` - Autoscaling triggers in use by this environment.



[1]: http://docs.aws.amazon.com/fr_fr/elasticbeanstalk/latest/dg/concepts.platforms.html
