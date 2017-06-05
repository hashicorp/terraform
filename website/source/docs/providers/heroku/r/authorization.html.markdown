---
layout: "heroku"
page_title: "Heroku: heroku_authorization"
sidebar_current: "docs-heroku-resource-authorization"
description: |-
  Provides a Heroku OAuth Authorization resource. This can be used to create and manage direct authorizations on Heroku.
---

# heroku\_authorization

Provides a Heroku OAuth Authorization resource. This can be used to
create and manage direct authorizations on Heroku.

## Example Usage

```hcl
# Create a new Heroku authorization
resource "heroku_authorization" "default" {
  description = "Example Identity Token"
  scope = [ "identity" ]
}
```

## Argument Reference

The following arguments are supported:

* `description` - (Optional) A human-readable description of the authorization.
* `scope` - (Optional) The [OAuth scopes](https://devcenter.heroku.com/articles/oauth#scopes) to grant.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the token.
* `token` - The secret OAuth token created by this resource.
