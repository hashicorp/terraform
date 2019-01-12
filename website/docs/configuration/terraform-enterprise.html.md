---
layout: "docs"
page_title: "Configuring Terraform Push"
sidebar_current: "docs-config-push"
description: |-
  Terraform's push command was a way to interact with the legacy version of Terraform Enterprise. It is not supported in the current version of Terraform Enterprise.
---

# Terraform Push Configuration

Prior to v0.12, Terraform included mechanisms to interact with a legacy version
of Terraform Enterprise, formerly known as "Atlas".

These features relied on a special configuration block named `atlas`:

```hcl
atlas {
  name = "acme-corp/production"
}
```

These features are no longer available on Terraform Enterprise and so the
corresponding configuration elements and commands have been removed in
Terraform v0.12.

To migrate to the current version of Terraform Enterprise, refer to
[the upgrade guide](/docs/enterprise/upgrade/index.html). After upgrading,
any `atlas` blocks in your configuration can be safely removed.
