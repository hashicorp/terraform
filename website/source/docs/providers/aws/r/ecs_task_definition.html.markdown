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

```hcl
resource "aws_ecs_task_definition" "service" {
  family                = "service"
  container_definitions = "${file("task-definitions/service.json")}"

  volume {
    name      = "service-storage"
    host_path = "/ecs/service-storage"
  }

  placement_constraints {
    type       = "memberOf"
    expression = "attribute:ecs.availability-zone in [us-west-2a, us-west-2b]"
  }
}
```

The referenced `task-definitions/service.json` file contains a valid JSON document,
which is shown below, and its content is going to be passed directly into the
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

* `family` - (Required) A unique name for your task definition.
* `container_definitions` - (Required) A list of valid [container definitions]
(http://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ContainerDefinition.html) provided as a
single valid JSON document. Please note that you should only provide values that are part of the container
definition document. For a detailed description of what parameters are available, see the [Task Definition Parameters]
(https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_definition_parameters.html) section from the
official [Developer Guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide).
* `task_role_arn` - (Optional) The ARN of IAM role that allows your Amazon ECS container task to make calls to other AWS services.
* `network_mode` - (Optional) The Docker networking mode to use for the containers in the task. The valid values are `none`, `bridge`, and `host`.
* `volume` - (Optional) A volume block. See below for details about what arguments are supported.
* `placement_constraints` - (Optional) rules that are taken into consideration during task placement. Maximum number of
`placement_constraints` is `10`. Defined below.

Volume block supports the following arguments:

* `name` - (Required) The name of the volume. This name is referenced in the `sourceVolume`
parameter of container definition in the `mountPoints` section.
* `host_path` - (Optional) The path on the host container instance that is presented to the container. If not set, ECS will create a nonpersistent data volume that starts empty and is deleted after the task has finished.

## placement_constraints

`placement_constraints` support the following:

* `type` - (Required) The type of constraint. Use `memberOf` to restrict selection to a group of valid candidates.
Note that `distinctInstance` is not supported in task definitions.
* `expression` -  (Optional) Cluster Query Language expression to apply to the constraint.
For more information, see [Cluster Query Language in the Amazon EC2 Container
Service Developer
Guide](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/cluster-query-language.html).


## Attributes Reference

The following attributes are exported:

* `arn` - Full ARN of the Task Definition (including both `family` and `revision`).
* `family` - The family of the Task Definition.
* `revision` - The revision of the task in a particular family.
