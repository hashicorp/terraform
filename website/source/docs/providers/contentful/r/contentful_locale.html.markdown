---
layout: "contentful"
page_title: "Contentful: contentful_locale"
sidebar_current: "docs-contentful-resource-contentful_locale"
description: |-
  Creates and manages a locales for a space.
---

# contentful\_locale

The ``contentful_locale`` resource creates and manages [locales](https://www.contentful.com/developers/docs/references/content-management-api/#/reference/locales).
This allow you to define translatable content to be delivered.

## Usage

```hcl
resource "contentful_space" "myspace" {
  name = "My space"
  default_locale = "en-US"
}

resource "contentful_locale" "mylocale" {
  space_id = "${contentful_space.myspace.id}"

  name = "german-locale"
  code = "de"
  fallback_code = "en-US"
  optional = false
  cda = false
  cma = true
}
```

## Argument Reference

* `space_id` - (Required) The space ID where the locale will be created.

* `name` - (Required) The name of the locale.

* `code` - (Required) Language code.

* `fallback_code` - (Required) If no content exists for a requested locale, the Delivery API will return content in this locale.

* `optional` - (Optional) The default locale used for new content types.

* `cda` - (Optional) The default locale used for new content types.

* `cma` - (Optional) The default locale used for new content types.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the locale.

* `version` - The version of the locale.
