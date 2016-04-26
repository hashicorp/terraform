---
layout: "aws"
page_title: "AWS: aws_db_event_subscription"
sidebar_current: "docs-aws-resource-db-event-subscription"
---

# aws\_db\_event\_subscription

Provides a DB event subscription resource.

## Example Usage

```
resource "aws_sns_topic" "default" {
  name = "rds-events"
}

resource "aws_db_event_subscription" "default" {
  name = "rds-event-sub"
  sns_topic = "${aws_sns_topic.default.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DB event subscription.
* `sns_topic` - (Required) The SNS topic to send events to.
* `source_ids` - (Optional) A list of identifiers of the event sources for which events will be returned. If not specified, then all sources are included in the response. If specified, a source_type must also be specified.
* `source_type` - (Optional) The type of source that will be generating the events.
* `event_categories` - (Optional) A list of event categories for a SourceType that you want to subscribe to.
* `enabled` - (Optional) A boolean flag to enable/disable the subscription. Defaults to true.
* `tags` - (Optional) A mapping of tags to assign to the resource.
