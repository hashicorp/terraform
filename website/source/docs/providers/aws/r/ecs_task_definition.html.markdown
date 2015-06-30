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
resource "aws_ecs_task_definition" "jenkins" {
  family = "jenkins"
  container_definitions = "${file("task-definitions/jenkins.json")}"

  volume {
    name = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }
}
```

## Argument Reference

The following arguments are supported:

* `family` - (Required) The family, unique name for your task definition.
* `container_definitions` - (Required) A list of container definitions in JSON format. See [AWS docs](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_defintions.html) for syntax.
* `volume` - (Optional) A volume block. Volumes documented below.

Volumes support the following:

* `name` - (Required) The name of the volume. This name is referenced in the `sourceVolume` parameter of container definition `mountPoints`.
* `host_path` - (Required) The path on the host container instance that is presented to the container.

## Attributes Reference

The following attributes are exported:

* `arn` - Full ARN of the task definition (including both `family` & `revision`)
* `family` - The family of the task definition.
* `revision` - The revision of the task in a particular family.
