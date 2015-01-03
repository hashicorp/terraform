---
layout: "heroku"
page_title: "Heroku: heroku_drain"
sidebar_current: "docs-heroku-resource-drain"
description: |-
  Provides a Heroku Drain resource. This can be used to create and manage Log Drains on Heroku.
---

# heroku\_drain

Provides a Heroku Drain resource. This can be used to
create and manage Log Drains on Heroku.

## Example Usage

```
resource "heroku_drain" "default" {
    app = "test-app"
    url = "syslog://terraform.example.com:1234"
}
```

## Argument Reference

The following arguments are supported:

* `url` - (Required) The URL for Heroku to drain your logs to.
* `app` - (Required) The Heroku app to link to.

## Attributes Reference

The following attributes are exported:

* `token` - The unique token for your created drain.

