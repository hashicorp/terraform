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

    config_vars {
        FOOBAR = "baz"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application. In Heroku, this is also the
   unique ID, so it must be unique and have a minimum of 3 characters.
* `region` - (Required) The region that the app should be deployed in.
* `stack` - (Optional) The application stack is what platform to run the application
   in.
* `config_vars` - (Optional) Configuration variables for the application.
     The config variables in this map are not the final set of configuration
     variables, but rather variables you want present. That is, other
     configuration variables set externally won't be removed by Terraform
     if they aren't present in this list.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the app. This is also the name of the application.
* `name` - The name of the application. In Heroku, this is also the
   unique ID.
* `stack` - The application stack is what platform to run the application
   in.
* `region` - The region that the app should be deployed in.
* `git_url` - The Git URL for the application. This is used for
   deploying new versions of the app.
* `web_url` - The web (HTTP) URL that the application can be accessed
   at by default.
* `heroku_hostname` - A hostname for the Heroku application, suitable
   for pointing DNS records.
* `all_config_vars` - A map of all of the configuration variables that
    exist for the app, containing both those set by Terraform and those
    set externally.
