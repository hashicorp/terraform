---
layout: "circonus"
page_title: "Circonus: circonus_metric"
sidebar_current: "docs-circonus-resource-circonus_metric"
description: |-
  Manages a Circonus metric.
---

# circonus\_metric

The ``circonus_metric`` resource creates and manages a
single [metric resource](https://login.circonus.com/resources/api/calls/metric)
that will be instantiated only once a referencing `circonus_check` has been
created.

## Usage

```hcl
resource "circonus_metric" "used" {
  name  = "_usage`0`_used"
  type  = "numeric"
  units = "qty"

  tags = {
    author = "terraform"
    source = "circonus"
  }
}
```

## Argument Reference

* `active` - (Optional) A boolean indicating if the metric is being filtered out
  at the `circonus_check`'s collector(s) or not.

* `name` - (Required) The name of the metric.  A `name` must be unique within a
  `circonus_check` and its meaning is `circonus_check.type` specific.

* `tags` - (Optional) A list of tags assigned to the metric.

* `type` - (Required) The type of metric.  This value must be present and can be
  one of the following values: `numeric`, `text`, `histogram`, `composite`, or
  `caql`.

* `unit` - (Optional) The unit of measurement for this `circonus_metric`.

## Import Example

`circonus_metric` supports importing resources.  Supposing the following
Terraform:

```hcl
provider "circonus" {
  alias = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}

resource "circonus_metric" "usage" {
  name = "_usage`0`_used"
  type = "numeric"
  unit = "qty"
  tags = { source = "circonus" }
}
```

It is possible to import a `circonus_metric` resource with the following command:

```
$ terraform import circonus_metric.usage ID
```

Where `ID` is a random, never before used UUID and `circonus_metric.usage` is
the name of the resource whose state will be populated as a result of the
command.
