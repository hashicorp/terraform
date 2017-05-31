---
layout: "aws"
page_title: "AWS: aws_ssm_patch_baseline"
sidebar_current: "docs-aws-resource-ssm-patch-baseline"
description: |-
  Provides an SSM Patch Baseline resource
---

# aws_ssm_patch_baseline

Provides an SSM Patch Baseline resource

~> **NOTE on Patch Baselines:** The `approved_patches` and `approval_rule` are 
both marked as optional fields, but the Patch Baseline requires that at least one
of them is specified.

## Example Usage

Basic usage using `approved_patches` only

```hcl
resource "aws_ssm_patch_baseline" "production" {
  name  = "patch-baseline"
  approved_patches = ["KB123456"]
}
```

Advanced usage, specifying patch filters

```hcl
resource "aws_ssm_patch_baseline" "production" {
  name  = "patch-baseline"
  description = "Patch Baseline Description"
  approved_patches = ["KB123456", "KB456789"]
  rejected_patches = ["KB987654"]
  global_filter {
    key = "PRODUCT"
    values = ["WindowsServer2008"]
  }
  global_filter {
    key = "CLASSIFICATION"
    values = ["ServicePacks"]
  }
  global_filter {
    key = "MSRC_SEVERITY"
    values = ["Low"]
  }
  approval_rule {
    approve_after_days = 7
    patch_filter {
      key = "PRODUCT"
      values = ["WindowsServer2016"]
    }
    patch_filter {
      key = "CLASSIFICATION"
      values = ["CriticalUpdates", "SecurityUpdates", "Updates"]
    }
    patch_filter {
      key = "MSRC_SEVERITY"
      values = ["Critical", "Important", "Moderate"]
    }
  }
  approval_rule {
    approve_after_days = 7
    patch_filter {
      key = "PRODUCT"
      values = ["WindowsServer2012"]
    }
  }
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the patch baseline.
* `description` - (Optional) The description of the patch baseline.
* `approved_patches` - (Optional) A list of explicitly approved patches for the baseline.
* `rejected_patches` - (Optional) A list of rejected patches.
* `global_filter` - (Optional) A set of global filters used to exclude patches from the baseline. Up to 4 global filters can be specified using Key/Value pairs. Valid Keys are `PRODUCT | CLASSIFICATION | MSRC_SEVERITY | PATCH_ID`.
* `approval_rule` - (Optional) A set of rules used to include patches in the baseline. up to 10 approval rules can be specified. Each approval_rule block requires the fields documented below.

The `approval_rule` block supports:

* `approve_after_days` - (Required) The number of days after the release date of each patch matched by the rule the patch is marked as approved in the patch baseline. Valid Range: 0 to 100.
* `patch_filter` - (Required) The patch filter group that defines the criteria for the rule. Up to 4 patch filters can be specified per approval rule using Key/Value pairs. Valid Keys are `PRODUCT | CLASSIFICATION | MSRC_SEVERITY | PATCH_ID`.


## Attributes Reference

The following attributes are exported:

* `id` - The ID of the patch baseline.