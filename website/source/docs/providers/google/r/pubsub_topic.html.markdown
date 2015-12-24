---
layout: "google"
page_title: "Google: google_pubsub_topic"
sidebar_current: "docs-google-pubsub-topic"
description: |-
  Creates a topic in Google's pubsub  queueing system
---

# google\_pubsub\_topic

Creates a topic in Google's pubsub queueing system.  For more information see
[the official documentation](https://cloud.google.com/pubsub/docs) and
[API](https://cloud.google.com/pubsub/reference/rest/v1/projects.topics).


## Example Usage

```
resource "google_pubsub_topic" "default" {
	name = "default-topic"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by pubsub.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
