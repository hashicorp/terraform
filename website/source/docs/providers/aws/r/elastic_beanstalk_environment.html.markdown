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

```hcl
resource "aws_elastic_beanstalk_application" "tftest" {
  name        = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name                = "tf-test-name"
  application         = "${aws_elastic_beanstalk_application.tftest.name}"
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
* `wait_for_ready_timeout` - (Default: `20m`) The maximum
  [duration](https://golang.org/pkg/time/#ParseDuration) that Terraform should
  wait for an Elastic Beanstalk Environment to be in a ready state before timing
  out.
* `poll_interval` – The time between polling the AWS API to
check if changes have been applied. Use this to adjust the rate of API calls
for any `create` or `update` action. Minimum `10s`, maximum `180s`. Omit this to
use the default behavior, which is an exponential backoff
* `version_label` - (Optional) The name of the Elastic Beanstalk Application Version
to use in deployment.
* `tags` – (Optional) A set of tags to apply to the Environment. **Note:** at
this time the Elastic Beanstalk API does not provide a programatic way of
changing these tags after initial application


## Option Settings

Some options can be stack-specific, check [AWS Docs](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/command-options-general.html)
for supported options and examples.

The `setting` and `all_settings` mappings support the following format:

* `namespace` - unique namespace identifying the option's associated AWS resource
* `name` - name of the configuration option
* `value` - value for the configuration option
* `resource` - (Optional) resource name for [scheduled action](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/command-options-general.html#command-options-general-autoscalingscheduledaction)

### Example With Options

```hcl
resource "aws_elastic_beanstalk_application" "tftest" {
  name        = "tf-test-name"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name                = "tf-test-name"
  application         = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux 2015.03 v2.0.3 running Go 1.4"

  setting {
    namespace = "aws:ec2:vpc"
    name      = "VPCId"
    value     = "vpc-xxxxxxxx"
  }

  setting {
    namespace = "aws:ec2:vpc"
    name      = "Subnets"
    value     = "subnet-xxxxxxxx"
  }
}
```

## Attributes Reference

The following attributes are exported:

* `id` - ID of the Elastic Beanstalk Environment.
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



[1]: https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/concepts.platforms.html


## Import

Elastic Beanstalk Environments can be imported using the `id`, e.g.

```
$ terraform import aws_elastic_beanstalk_environment.prodenv e-rpqsewtp2j
```
