---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_addon"
sidebar_current: "docs-pagerduty-resource-addon"
description: |-
  Creates and manages an add-on in PagerDuty.
---

# pagerduty\_addon

With [add-ons](https://v2.developer.pagerduty.com/v2/page/api-reference#!/Add-ons/get_addons), third-party developers can write their own add-ons to PagerDuty's UI. Given a configuration containing a src parameter, that URL will be embedded in an iframe on a page that's available to users from a drop-down menu.

## Example Usage

```hcl
resource "pagerduty_addon" "example" {
  name = "Internal Status Page"
  src  = "https://intranet.example.com/status"
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Required) The name of the add-on.
  * `src` - (Required) The source URL to display in a frame in the PagerDuty UI. `HTTPS` is required.

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the add-on.

## Import

Add-ons can be imported using the `id`, e.g.

```
$ terraform import pagerduty_addon.example P3DH5M6
```
