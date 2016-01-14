---
layout: "google"
page_title: "Google: google_pubsub_subscription"
sidebar_current: "docs-google-pubsub-subscription"
description: |-
  Creates a subscription in Google's pubsub  queueing system
---

# google\_pubsub\_subscripion

Creates a subscription in Google's pubsub queueing system.  For more information see
[the official documentation](https://cloud.google.com/pubsub/docs) and
[API](https://cloud.google.com/pubsub/reference/rest/v1/projects.subscriptions).


## Example Usage

```
resource "google_pubsub_subscription" "default" {
    name = "default-subscription"
    topic = "default-topic"
    ack_deadline_seconds = 20
    push_config {
        endpoint = "https://example.com/push"
        attributes {
            x-goog-version = "v1"
        }
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by pubsub.
    Changing this forces a new resource to be created.

* `topic` - (Required) A topic to bind this subscription to, required by pubsub.
    Changing this forces a new resource to be created.

* `ack_deadline_seconds` - (Optional) The maximum number of seconds a
    subscriber has to acknowledge a received message, otherwise the message is
    redelivered. Changing this forces a new resource to be created.

The optional `push_config` block supports:

* `push_endpoint` - (Optional) The URL of the endpoint to which messages should
    be pushed. Changing this forces a new resource to be created.

* `attributes` - (Optional) Key-value pairs of API supported attributes used
    to control aspects of the message delivery. Currently, only
    `x-goog-version` is supported, which controls the format of the data
    delivery. For more information, read [the API docs
    here](https://cloud.google.com/pubsub/reference/rest/v1/projects.subscriptions#PushConfig.FIELDS.attributes).
    Changing this forces a new resource to be created.
