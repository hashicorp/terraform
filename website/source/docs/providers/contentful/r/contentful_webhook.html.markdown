---
layout: "contentful"
page_title: "Contentful: contentful_webhook"
sidebar_current: "docs-contentful-resource-contentful_webhook"
description: |-
  Creates and manages a webhook for a given organization.
---

# contentful\_webhook

The ``contentful_webhook`` resource creates and manages [webhooks](https://www.contentful.com/developers/docs/references/content-management-api/#/reference/webhooks).
Webhooks notify a person or service when content has changed.

## Usage

```hcl
resource "contentful_space" "myspace" {
  name = "My space"
  default_locale = "en-US"
}

resource "contentful_webhook" "mywebhook" {
  space_id = "${contentful_space.myspace.id}"

  name = "webhook-name"
  url=  "https://www.example.com/test"
  topics = [
    "Entry.create",
    "ContentType.create",
  ]
  headers {
    header1 = "header1-value"
    header2 = "header2-value"
  }
  http_basic_auth_username = "username"
  http_basic_auth_password = "password"
}
```

## Argument Reference

* `space_id` - (Required) The space ID where the webhook will be created.

* `name` - (Required) The name of the webhook.

* `url` - (Required) HTTP endpoint that will be called to deliver the notification.

* `http_basic_auth_username` - (Required) Username to be used if basic auth is configured on the endpoint.

* `http_basic_auth_password` - (Required) Password to be used if basic auth is configured on the endpoint.

* `headers` - (Required) A map of headers and it's values to be sent on the notification.

* `topics` - (Required) List of strings.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the webhook.

* `version` - The version of the webhook.
