---
layout: "contentful"
page_title: "Contentful: contentful_space"
sidebar_current: "docs-contentful-resource-contentful_space"
description: |-
  Creates and manages a contentful space for a given organization.
---

# contentful\_space

The ``contentful_space`` resource creates and manages [spaces](https://www.contentful.com/developers/docs/references/content-management-api/#/reference/spaces).
Spaces is where you define your content model and where your content exists.

## Usage

```hcl
resource "contentful_space" "myspace" {
  name = "My space"
  default_locale = "en-US"
}
```

## Argument Reference

* `name` - (Required) The name of the space.

* `default_locale` - (Optional) The default locale used for new content types.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the space.

* `version` - The version of the space.
