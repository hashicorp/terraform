---
layout: "enterprise"
page_title: "Managing Versions - Artifacts - Terraform Enterprise"
sidebar_current: "docs-enterprise-artifacts-versions"
description: |-
  Artifacts are versioned and assigned a version number, here is how to manage the versions.
---

# Managing Artifact Versions

Artifacts stored in Terraform Enterprise are versioned and assigned a version
number. Versions are useful to roll back, audit and deploy images specific
versions of images to certain environments in a targeted way.

This assumes you are familiar with the [artifact provider](https://terraform.io/docs/providers/terraform-enterprise/index.html)
in Terraform.

### Finding the Version of an Artifact

Artifact versions can be found with the [`terraform show` command](https://terraform.io/docs/commands/show.html),
or by looking at the Packer logs generated during builds. After a
successful artifact upload, version numbers are displayed. "latest" can
be used to use the latest version of the artifact.

The following output is from `terraform show`.

```text
atlas_artifact.web-worker:
  id = us-east-1:ami-3a0a1d52
  build = latest
  metadata_full.# = 1
  metadata_full.region-us-east-1 = ami-3a0a1d52
  name = my-username/web-worker
  slug = my-username/web-worker/amazon.image/7
  type = amazon.image
```

In this case, the version is 7 and can be found in the persisted slug
attribute.

### Pinning Artifacts to Specific Versions

You can pin artifacts to a specific version. This allows for a targeted
deploy.

```hcl
data "atlas_artifact" "web-worker" {
  name  = "my-username/web-worker"
  type  = "amazon.image"
  version = 7
}
```

This will use version 7 of the `web-worker` artifact.

### Pinning Artifacts to Specific Builds

Artifacts can also be pinned to an Terraform build number. This is only
possible if Terraform Enterprise was used to build the artifact with Packer.

```hcl
data "atlas_artifact" "web-worker" {
  name  = "my-username/web-worker"
  type  = "amazon.image"
  build = 5
}
```

It's recommended to use versions, instead of builds, as it will be easier to
track when building outside of the Terraform Enterprise environment.
