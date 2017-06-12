---
layout: "google"
page_title: "Google: google_appengine_app"
sidebar_current: "docs-google-appengine-app"
description: |-
  Creates an App Engine app
---

# google\_appengine\_app

Creates a Google Ap Engine app. For more information see
[the official documentation](https://cloud.google.com/appengine/docs/) and
[API](https://cloud.google.com/appengine/docs/admin-api/reference/rest/).


## Example Usage

```hcl
resource "google_appengine_app" "default" {
  project = "my-google-project"
  region = "us-central"
}
```

## Argument Reference

The following arguments are supported:

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) The region where the App Engine app will be deployed. This is bound to the
    project and cannot be changed for the lifetime of the project.

## Attributes Reference

Only the arguments listed above are exposed as attributes.
