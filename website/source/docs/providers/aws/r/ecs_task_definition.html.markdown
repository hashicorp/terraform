---
layout: "aws"
page_title: "AWS: aws_ecs_task_definition"
sidebar_current: "docs-aws-resource-ecs-task-definition"
description: |-
  Provides an ECS task definition.
---

# aws\_ecs\_task\_definition

Provides an ECS task definition to be used in `aws_ecs_service`.

## Example Usage

```
resource "aws_ecs_task_definition" "service" {
  family = "service"
  container_definitions = "${file("task-definitions/service.json")}"

  volume {
    name = "service-storage"
    host_path = "/ecs/service-storage"
  }
}
```

The referenced `task-definitions/service.json` file contains a valid JSON document,
which is show below, and its content is going to be passed directly into the
`container_definitions` attribute as a string. Please note that this example
contains only a small subset of the available parameters.

```
[
  {
    "name": "first",
    "image": "service-first",
    "cpu": 10,
    "memory": 512,
    "essential": true,
    "portMappings": [
      {
        "containerPort": 80,
        "hostPort": 80
      }
    ]
  },
  {
    "name": "second",
    "image": "service-second",
    "cpu": 10,
    "memory": 256,
    "essential": true,
    "portMappings": [
      {
        "containerPort": 443,
        "hostPort": 443
      }
    ]
  }
]
```

## Argument Reference

The following arguments are supported:

* `family` - (Required) An unique name for your task definition.
* `container_definitions` - (Required) A list of valid [container definitions]
(http://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ContainerDefinition.html) provided as a
single valid JSON document. Please note that you should only provide values that are part of the container
definition document. For a detailed description of what parameters are available, see the [Task Definition Parameters]
(https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_definition_parameters.html) section from the
official [Developer Guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide).
* `task_role_arn` - (Optional) The ARN of IAM role that allows your Amazon ECS container task to make calls to other AWS services.
* `network_mode` - (Optional) The Docker networking mode to use for the containers in the task. The valid values are `none`, `bridge`, and `host`.
* `volume` - (Optional) A volume block. See below for details about what arguments are supported.

Volume block supports the following arguments:

* `name` - (Required) The name of the volume. This name is referenced in the `sourceVolume`
parameter of container definition in the `mountPoints` section.
* `host_path` - (Required) The path on the host container instance that is presented to the container.

## Attributes Reference

The following attributes are exported:

* `arn` - Full ARN of the Task Definition (including both `family` and `revision`).
* `family` - The family of the Task Definition.
* `revision` - The revision of the task in a particular family.
