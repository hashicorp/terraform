---
layout: "aws"
page_title: "AWS: aws_ssm_patch_group"
sidebar_current: "docs-aws-resource-ssm-patch-group"
description: |-
  Provides an SSM Patch Group resource
---

# aws_ssm_patch_group

Provides an SSM Patch Group resource

## Example Usage

```hcl
resource "aws_ssm_patch_baseline" "production" {
  name  = "patch-baseline"
  approved_patches = ["KB123456"]
}

resource "aws_ssm_patch_group" "patchgroup" {
  baseline_id = "${aws_ssm_patch_baseline.production.id}"
  patch_group = "patch-group-name"
}```

## Argument Reference

The following arguments are supported:

* `baseline_id` - (Required) The ID of the patch baseline to register the patch group with.
* `patch_group` - (Required) The name of the patch group that should be registered with the patch baseline.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the patch baseline.