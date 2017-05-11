---
layout: "google"
page_title: "Google: google_pubsub_topic"
sidebar_current: "docs-google-pubsub-topic"
description: |-
  Creates a topic in Google's pubsub  queueing system
---

# google\_pubsub\_topic

Creates a topic in Google's pubsub queueing system. For more information see
[the official documentation](https://cloud.google.com/pubsub/docs) and
[API](https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.topics).


## Example Usage

```hcl
resource "google_pubsub_topic" "default" {
  name = "default-topic"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by pubsub.
    Changing this forces a new resource to be created.

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

Only the arguments listed above are exposed as attributes.
