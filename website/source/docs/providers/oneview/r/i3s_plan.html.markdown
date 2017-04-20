---
layout: "oneview"
page_title: "Oneview: i3s_plan"
sidebar_current: "docs-oneview-i3s-plan"
description: |-
  Adds a deployment plan to a server.
---

# oneview\_i3s\_plan

Adds a deployment plan to a server.

## Example Usage

```js
resource "oneview_i3s_plan" "default" {
  server_name = "${oneview_server_profile.default.name}"
  os_deployment_plan = "Ubuntu 16.04"
}
```

## Argument Reference

The following arguments are supported: 

* `server_name` - (Required) The name of the server that the deployment plan will be run on.

* `os_deployment_plan` - (Required) The name of the deployment plan that will run on the server. 

- - -

* `deployment_attribute` - (Optional) A key/value pair that modifies the default values provided by the os deployment plan
  This can be specified multiple times. Deployment Attribute is described below.
  
Deployment Attribute supports the following:

* `key` - (Required) - The unique name of the attribute.

* `value` - (Required) - The value of the attribute.


