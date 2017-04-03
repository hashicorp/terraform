---
layout: "contentful"
page_title: "Contentful: contentful_api"
sidebar_current: "docs-contentful-resource-contentful_api"
description: |-
  Creates and manages content delivery API keys for a given space.
---

# contentful\_api

The ``contentful_api`` resource creates and manages content delivery [API keys](https://www.contentful.com/developers/docs/references/content-management-api/#/reference/api-keys).
These tokens provide read-only access to a single space.

## Usage

```hcl
resource "contentful_space" "myspace" {
  name = "My space"
  default_locale = "en-US"
}

resource "contentful_apikey" "myapikey" {
  space_id = "${contentful_space.myspace.id}"

  name = "apikey-name"
  description = "apikey-description"
}
```

## Argument Reference

* `space_id` - (Required) The space ID which the API key will allow read-only access to.

* `name` - (Required) The name of the API key.

* `description` - (Optional) A description that will better identification of this key.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the API key.

* `version` - The version of the API key.
