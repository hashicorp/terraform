---
layout: "contentful"
page_title: "Provider: Contentful"
sidebar_current: "docs-contentful-index"
description: |-
  A provider for Contentful.
---

# Contentful Provider

[`Contentful`](https://www.contentful.com) is a content management platform for web applications, mobile apps and connected devices. It allows you to create, edit & manage content in the cloud and publish it anywhere via a powerful RESTful JSON API.
The Contentful provider allows the development and deployment of those resources.

Use the navigation to the left for more information about the available resources.

## Example Usage

```hcl
# Configure Contentful's provider
provider "contentful" {
  cma_token       = "<your CMA Token>"
  organization_id = "<your organization ID>"
}

# Create a blog space
resource "contentful_space" "myblog" {
  name = "My Blog"
}

# Create a content type to hold the blog posts
resource "contentful_contenttype" "post" {
  space_id = "${contentful_space.myblog.id}"

  name = "Post"
  description = "My blog posts"
  display_field = "title"

  field {
    id = "title"
    name = "Title"
    type = "Text"
    required = true
  }

  field {
    id = "body"
    name = "Body"
    type = "Text"
    required = true
  }

  field {
    id = "tags"
    name = "Tags"
    type = "Array"
    items {
      type = "Symbol"
    }
    required = false
  }

}
```

## Argument Reference

The following arguments are supported:

* `cma_token` - (Required) This is the token that is authorized to make Content Management API calls.
* `organization_id` - (Required) The organization owning the resources.

Both the api token and the organization id can be obtained by creating an account at [`Contentful`](https://www.contentful.com/sign-up).

These can also be provided via environment variables:

* CONTENTFUL_MANAGEMENT_TOKEN
* CONTENTFUL_ORGANIZATION_ID
