---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-resource-app"
---

# heroku\_app

Provides a Heroku App resource. This can be used to
create and manage applications on Heroku.

## Example Usage

```
# Create a new heroku app
resource "heroku_app" "default" {
    name = "my-cool-app"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the Heroku app
* `region` - (Optional) The region of the Heroku app
* `stack` - (Optional) The stack for the Heroku app
* `config_vars` - (Optional) Configuration variables for the app

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the app
* `name` - The name of the app
* `stack` - The stack of the app
* `region` - The region of the app
* `git_url` - The Git URL for the app
* `web_url` - The Web URL for the app
* `heroku_hostname` - The Heroku URL for the app

