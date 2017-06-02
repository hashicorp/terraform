---
layout: "enterprise"
page_title: "Provider - Artifacts - Terraform Enterprise"
sidebar_current: "docs-enterprise-artifacts-provider"
description: |-
  Terraform has a provider for managing artifacts called `atlas_artifact`.
---

# Artifact Provider

Terraform has a [provider](https://terraform.io/docs/providers/index.html) for managing Terraform Enterprise artifacts called `atlas_artifact`.

This is used to make data stored in Artifacts available to Terraform for
interpolation. In the following example, an artifact is defined and references
an AMI ID stored in Terraform Enterprise.

~> **Why is this called "atlas"?** Atlas was previously a commercial offering
from HashiCorp that included a full suite of enterprise products. The products
have since been broken apart into their individual products, like **Terraform
Enterprise**. While this transition is in progress, you may see references to
"atlas" in the documentation. We apologize for the inconvenience.

```hcl
provider "atlas" {
  # You can also set the atlas token by exporting ATLAS_TOKEN into your env
  token = "${var.atlas_token}"
}

data "atlas_artifact" "web-worker" {
  name    = "my-username/web-worker"
  type    = "amazon.image"
  version = "latest"
}

resource "aws_instance" "worker-machine" {
  ami           = "${atlas_artifact.web-worker.metadata_full.region-us-east-1}"
  instance_type = "m1.small"
}
```

This automatically pulls the "latest" artifact version.

Following a new artifact version being created via a Packer build, the following
diff would be generated when running `terraform plan`.

```
-/+ aws_instance.worker-machine
    ami:             "ami-168f9d7e" => "ami-2f3a9df2" (forces new resource)
    instance_type:   "m1.small" => "m1.small"
```

This allows you to reference changing artifacts and trigger new deployments upon
pushing subsequent Packer builds.

Read more about artifacts in the [Terraform documentation](https://terraform.io/docs/providers/terraform-enterprise/r/artifact.html).
