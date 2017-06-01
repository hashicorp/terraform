---
layout: "aws"
page_title: "AWS: aws_lightsail_instance"
sidebar_current: "docs-aws-resource-lightsail-instance"
description: |-
  Provides an Lightsail Instance
---

# aws\_lightsail\_instance

Provides a Lightsail Instance. Amazon Lightsail is a service to provide easy virtual private servers
with custom software already setup. See [What is Amazon Lightsail?](https://lightsail.aws.amazon.com/ls/docs/getting-started/article/what-is-amazon-lightsail)
for more information.

~> **Note:** Lightsail is currently only supported in a limited number of AWS Regions, please see ["Regions and Availability Zones in Amazon Lightsail"](https://lightsail.aws.amazon.com/ls/docs/overview/article/understanding-regions-and-availability-zones-in-amazon-lightsail) for more details

## Example Usage

```hcl
# Create a new GitLab Lightsail Instance
resource "aws_lightsail_instance" "gitlab_test" {
  name              = "custom gitlab"
  availability_zone = "us-east-1b"
  blueprint_id      = "string"
  bundle_id         = "string"
  key_pair_name     = "some_key_name"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Lightsail Instance
* `availability_zone` - (Required) The Availability Zone in which to create your
instance. At this time, must be in `us-east-1`, `us-east-2`, `us-west-2`, `eu-west-1`, `eu-west-2`, `eu-central-1` regions
* `blueprint_id` - (Required) The ID for a virtual private server image
(see list below)
* `bundle_id` - (Required) The bundle of specification information (see list below)
* `key_pair_name` - (Required) The name of your key pair. Created in the
Lightsail console (cannot use `aws_key_pair` at this time)
* `user_data` - (Optional) launch script to configure server with additional user data


## Blueprints

Lightsail currently supports the following Blueprint IDs:

- `amazon_linux_2016_09_0`
- `ubuntu_16_04`
- `wordpress_4_6_1`
- `lamp_5_6_27`
- `nodejs_6_9_1`
- `joomla_3_6_3`
- `magento_2_1_2`
- `mean_3_2_10`
- `drupal_8_2_1`
- `gitlab_8_12_6`
- `redmine_3_3_1`
- `nginx_1_10_2`

## Bundles

Lightsail currently supports the following Bundle IDs:

- `nano_1_0`
- `micro_1_0`
- `small_1_0`
- `medium_1_0`
- `large_1_0`

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The ARN of the Lightsail instance (matches `arn`).
* `arn` - The ARN of the Lightsail instance (matches `id`).
* `availability_zone`
* `blueprint_id`
* `bundle_id`
* `key_pair_name`
* `user_data`

## Import

Lightsail Instances can be imported using their ARN, e.g.

```
$ terraform import aws_lightsail_instance.bar <arn>
```
