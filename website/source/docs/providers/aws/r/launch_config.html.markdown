---
layout: "aws"
page_title: "AWS: aws_launch_configuration"
sidebar_current: "docs-aws-resource-launch-config"
---

# aws\_launch\_configuration

Provides a resource to create a new launch configuration, used for autoscaling groups.

## Example Usage

```
resource "aws_launch_configuration" "as_conf" {
    name = "web_config"
    image_id = "ami-1234"
    instance_type = "m1.small"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the launch configuration.
* `image_id` - (Required) The EC2 image ID to launch.
* `instance_type` - (Required) The size of instance to launch.
* `key_name` - (Optional) The key name that should be used for the instance.
* `security_groups` - (Optional) A list of associated security group IDS.
* `user_data` - (Optional) The user data to provide when launching the instance.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the launch configuration.
