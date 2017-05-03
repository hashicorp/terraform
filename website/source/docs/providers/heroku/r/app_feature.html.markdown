---
layout: "heroku"
page_title: "Heroku: heroku_app_feature"
sidebar_current: "docs-heroku-resource-app-feature"
description: |-
  Provides a Heroku App Feature resource. This can be used to create and manage App Features on Heroku.
---

# heroku\_app\_feature

Provides a Heroku App Feature resource. This can be used to create and manage App Features on Heroku.

## Example Usage

```hcl
resource "heroku_app_feature" "log_runtime_metrics" {
  app = "test-app"
  name = "log-runtime-metrics"
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The Heroku app to link to.
* `name` - (Required) The name of the App Feature to manage.
* `enabled` - (Optional) Whether to enable or disable the App Feature. The default value is true.
