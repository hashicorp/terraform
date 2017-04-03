---
layout: "contentful"
page_title: "Contentful: contentful_contenttype"
sidebar_current: "docs-contentful-resource-contentful_contenttype"
description: |-
  Creates and manages the content types in a space.
---

# contentful\_contenttype

The ``contentful_contenttype`` resource creates and manages [content types](https://www.contentful.com/developers/docs/references/content-management-api/#/reference/content-types).

## Usage

```hcl
resource "contentful_space" "myspace" {
  name = "Product Catalog"
}

resource "contentful_contenttype" "brand" {
  space_id = "${contentful_space.myspace.id}"

  name = "Brand"
  description = "Brands making our products"
  display_field = "companyName"

  field {
    id = "companyName"
    name = "Company Name"
    type = "Text"
    required = true
  }

  field {
    id = "logo"
    name = "Logo"
    type = "Link"
    link_type = "Asset"
    required = false
  }
}

resource "contentful_contenttype" "product" {
  space_id = "${contentful_space.myspace.id}"

  name = "Product"
  description = "Products we sell"
  display_field = "productName"

  field {
    id = "productName"
    name = "Product Name"
    type = "Text"
    required = true
  }

  field {
    id = "slug"
    name = "Slug"
    type = "Symbol"
    required = false
  }

  field {
    id = "image"
    name = "Image"
    type = "Array"
		items {
			type = "Link"
			link_type = "Asset"
		}
    required = false
  }

  field {
    id = "brand"
    name = "Brand"
    type = "Link"
		link_type = "Entry"
    validations = ["<<VALIDATION
{
  "linkContentType": ["${contentful_contenttype.brand.id}"]
}
VALIDATION
    "]
    required = false
  }
}
```

## Argument Reference

* `space_id` - (Required) The space ID where the content type will be created..

* `name` - (Required) The name of the content type.

* `description` - (Required) The name of the content type.

* `display_field` - (Required) The name of the content type.

* `field` - (Required) A list of field objects. Their keys are documented bellow.

Each field supports the following:

* `id` - (Required) The id of the field.

* `name` - (Required) The name of field.

* `type` - (Required) The type the field holds. In the case of Array fields the items object becomes mandatory. More info about Arrays [here](https://www.contentful.com/developers/docs/concepts/data-model/#array-fields).

* `link_type` - (Required) If the field is of type Link, which kind of link is it. More informations about Links [here](https://www.contentful.com/developers/docs/concepts/links/).

* `required` - (Required) Determines if this field can be left empty.

* `localized` - (Required) Is the field localized.

* `disabled` - (Required) Indicates if the field is disabled.

* `omitted` - (Required) Indicates if the field is omitted, thus not served in delivery requests.

* `validations` - (Required) A list of validations in the form of JSON. More details [here](https://www.contentful.com/developers/docs/references/content-management-api/#/reference/content-types)

* `items` - (Required) A list of item objects when a field is of type Array. This property defines the allowed values in the Array. Their keys are documented bellow.

Each field supports the following:

* `type` - (Required) The type this array item holds.

* `validations` - (Required) A list of validations in the form of JSON associated to this array item.

* `link_type` - (Required) If this array item is of type Link, which kind of link is it.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the content type.

* `version` - The version of the content type.

