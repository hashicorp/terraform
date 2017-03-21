---
title: "About Packer and Artifacts "
---

# About Packer and Artifacts

Packer creates and uploads artifacts to Atlas. This is done
with the [Atlas post-processor](https://packer.io/docs/post-processors/atlas.html).

Artifacts can then be used in Atlas to deploy services or access
via Vagrant. Artifacts are generic, but can be of varying types.
These types define different behavior within Atlas.

For uploading artifacts to Atlas, `artifact_type` can be set to any
unique identifier, however, the following are recommended for consistency.

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

Packer can create artifacts when running in Atlas or locally.
This is possible due to the post-processors use of the public
artifact API to store the artifacts.

You can read more about artifacts and their use in the [Terraform section](/help/terraform/features)
of the documentation.
