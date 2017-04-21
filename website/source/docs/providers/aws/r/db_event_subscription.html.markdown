---
layout: "aws"
page_title: "AWS: aws_db_event_subscription"
sidebar_current: "docs-aws-resource-db-event-subscription"
---

# aws\_db\_event\_subscription

Provides a DB event subscription resource.

## Example Usage

```hcl
resource "aws_db_instance" "default" {
  allocated_storage    = 10
  engine               = "mysql"
  engine_version       = "5.6.17"
  instance_class       = "db.t1.micro"
  name                 = "mydb"
  username             = "foo"
  password             = "bar"
  db_subnet_group_name = "my_database_subnet_group"
  parameter_group_name = "default.mysql5.6"
}

resource "aws_sns_topic" "default" {
  name = "rds-events"
}

resource "aws_db_event_subscription" "default" {
  name      = "rds-event-sub"
  sns_topic = "${aws_sns_topic.default.arn}"

  source_type = "db-instance"
  source_ids  = ["${aws_db_instance.default.id}"]

  event_categories = [
    "availability",
    "deletion",
    "failover",
    "failure",
    "low storage",
    "maintenance",
    "notification",
    "read replica",
    "recovery",
    "restoration",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DB event subscription.
* `sns_topic` - (Required) The SNS topic to send events to.
* `source_ids` - (Optional) A list of identifiers of the event sources for which events will be returned. If not specified, then all sources are included in the response. If specified, a source_type must also be specified.
* `source_type` - (Optional) The type of source that will be generating the events.
* `event_categories` - (Optional) A list of event categories for a SourceType that you want to subscribe to. See http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide//USER_Events.html
* `enabled` - (Optional) A boolean flag to enable/disable the subscription. Defaults to true.
* `tags` - (Optional) A mapping of tags to assign to the resource.


## Import

DB Event Subscriptions can be imported using the `name`, e.g.

```
$ terraform import aws_db_event_subscription.default rds-event-sub
```
