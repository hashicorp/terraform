---
layout: "aws"
page_title: "AWS: aws_ecs_service"
sidebar_current: "docs-aws-resource-ecs-service"
description: |-
  Provides an ECS service.
---

# aws\_ecs\_service

-> **Note:** To prevent race condition during service deletion, make sure to set `depends_on` to related `aws_iam_role_policy`, otherwise policy may be destroyed too soon and ECS service will then stuck in `DRAINING` state.

Provides an ECS service - effectively a task that is expected to run until an error occures or user terminates it (typically a webserver or a database).

See [ECS Services section in AWS developer guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_services.html).

## Example Usage

```
resource "aws_ecs_service" "mongo" {
  name = "mongodb"
  cluster = "${aws_ecs_cluster.foo.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 3
  iam_role = "${aws_iam_role.foo.arn}"
  depends_on = ["aws_iam_role_policy.foo"]

  load_balancer {
    elb_name = "${aws_elb.foo.name}"
    container_name = "mongo"
    container_port = 8080
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the service (up to 255 letters, numbers, hyphens, and underscores)
* `task_definition` - (Required) The family and revision (`family:revision`) or full ARN of the task definition that you want to run in your service.
* `desired_count` - (Required) The number of instances of the task definition to place and keep running
* `cluster` - (Optional) ARN of an ECS cluster
* `iam_role` - (Optional) IAM role that allows your Amazon ECS container agent to make calls to your load balancer on your behalf. This parameter is only required if you are using a load balancer with your service.
* `deployment_maximum_percent` - (Optional) The upper limit (as a percentage of the service's desiredCount) of the number of running tasks that can be running in a service during a deployment.
* `deployment_minimum_healthy_percent` - (Optional) The lower limit (as a percentage of the service's desiredCount) of the number of running tasks that must remain running and healthy in a service during a deployment.
* `load_balancer` - (Optional) A load balancer block. Load balancers documented below.

-> **Note:** As a result of AWS limitation a single `load_balancer` can be attached to the ECS service at most. See [related docs](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-load-balancing.html#load-balancing-concepts).

Load balancers support the following:

* `elb_name` - (Required) The name of the load balancer.
* `container_name` - (Required) The name of the container to associate with the load balancer (as it appears in a container definition).
* `container_port` - (Required) The port on the container to associate with the load balancer.


## Attributes Reference

The following attributes are exported:

* `id` - The Amazon Resource Name (ARN) that identifies the service
* `name` - The name of the service
* `cluster` - The Amazon Resource Name (ARN) of cluster which the service runs on
* `iam_role` - The ARN of IAM role used for ELB
* `desired_count` - The number of instances of the task definition
