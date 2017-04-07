---
layout: "enterprise"
page_title: "Packer Artifacts - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerartifacts"
description: |-
  Packer creates and uploads artifacts to Terraform Enterprise.
---

# About Packer and Artifacts

Packer creates and uploads artifacts to Terraform Enterprise. This is done
with the [post-processor](https://packer.io/docs/post-processors/atlas.html).

Artifacts can then be used to deploy services or access via Vagrant. Artifacts
are generic, but can be of varying types. These types define different behavior
within Terraform Enterprise.

For uploading artifacts `artifact_type` can be set to any unique identifier,
however, the following are recommended for consistency.

- `amazon.image`
- `azure.image`
- `digitalocean.image`
- `docker.image`
- `google.image`
- `openstack.image`
- `parallels.image`
- `qemu.image`
- `virtualbox.image`
- `vmware.image`
- `custom.image`
- `application.archive`
- `vagrant.box`

Packer can create artifacts when running in Terraform Enterprise or locally.
This is possible due to the post-processors use of the public artifact API to
store the artifacts.

You can read more about artifacts and their use in the
[Terraform section](/docs/enterprise/) of the documentation.
