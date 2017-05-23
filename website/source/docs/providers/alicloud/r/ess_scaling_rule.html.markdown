---
layout: "alicloud"
page_title: "Alicloud: alicloud_ess_scaling_rule"
sidebar_current: "docs-alicloud-resource-ess-scaling-rule"
description: |-
  Provides a ESS scaling rule resource.
---

# alicloud\_ess\_scaling\_rule

Provides a ESS scaling rule resource.

## Example Usage

```
resource "alicloud_ess_scaling_group" "scaling" {
  # Other parameters...
}

resource "alicloud_ess_scaling_configuration" "config" {
  # Other parameters...
}

resource "alicloud_ess_scaling_rule" "rule" {
  scaling_group_id = "${alicloud_ess_scaling_group.scaling.id}"
  adjustment_type  = "TotalCapacity"
  adjustment_value = 2
  cooldown         = 60
}
```

## Argument Reference

The following arguments are supported:

* `scaling_group_id` - (Required) ID of the scaling group of a scaling rule.
* `adjustment_type` - (Required) Adjustment mode of a scaling rule. Optional values:
    - QuantityChangeInCapacity: It is used to increase or decrease a specified number of ECS instances.
    - PercentChangeInCapacity: It is used to increase or decrease a specified proportion of ECS instances.
    - TotalCapacity: It is used to adjust the quantity of ECS instances in the current scaling group to a specified value.
* `adjustment_value` - (Required) Adjusted value of a scaling rule. Value range:
    - QuantityChangeInCapacity：(0, 100] U (-100, 0]
    - PercentChangeInCapacity：[0, 10000] U [-10000, 0]
    - TotalCapacity：[0, 100]
* `scaling_rule_name` - (Optional) Name shown for the scaling rule, which is a string containing 2 to 40 English or Chinese characters.
* `cooldown` - (Optional) Cool-down time of a scaling rule. Value range: [0, 86,400], in seconds. The default value is empty.


## Attributes Reference

The following attributes are exported:

* `id` - The scaling rule ID.
* `scaling_group_id` - The id of scaling group.
* `ari` - Unique identifier of a scaling rule.
* `adjustment_type` - Adjustment mode of a scaling rule.
* `adjustment_value` - Adjustment value of a scaling rule.
* `scaling_rule_name` - Name of a scaling rule.
* `cooldown` - Cool-down time of a scaling rule.