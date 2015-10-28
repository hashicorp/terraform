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
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by pubsub.
    Changing this forces a new resource to be created.
* `topic` - (Required) A topic to bind this subscription to, required by pubsub.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `topic` - The topic to bind this resource to.
