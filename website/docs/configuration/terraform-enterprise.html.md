---
layout: "docs"
page_title: "Configuring Terraform Push"
sidebar_current: "docs-config-push"
description: |-
  Terraform's push command was a way to interact with the legacy version of Terraform Enterprise. It is not supported in the current version of Terraform Enterprise.
---

# Terraform Push Configuration

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Configuring Terraform Push](../configuration-0-11/terraform-enterprise.html).

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

After upgrading to the current version of Terraform Enterprise,
any `atlas` blocks in your configuration can be safely removed.
