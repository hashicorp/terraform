---
layout: "aws"
page_title: "AWS: ses_configuration_set"
sidebar_current: "docs-aws-resource-ses-configuration-set"
description: |-
  Provides an SES configuration set
---

# aws\_ses\_configuration_set

Provides an SES configuration set resource

## Example Usage

```hcl
resource "aws_ses_configuration_set" "test" {
  name = "some-configuration-set-test"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the configuration set
