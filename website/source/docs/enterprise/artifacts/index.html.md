---
title: "About Terraform Artifacts in Atlas"
---

# About Terraform Artifacts in Atlas

Atlas can be used to store artifacts for use by Terraform. Typically,
artifacts are [stored with Packer](/help/packer/artifacts).

Artifacts can be used in Atlas to deploy and manage images
of configuration. Artifacts are generic, but can be of varying types
like `amazon.image`. See the Packer [`artifact_type`](https://packer.io/docs/post-processors/atlas.html#artifact_type)
docs for more information.

Packer can create artifacts both while running in Atlas and out of Atlas'
network. This is possible due to the post-processors use of the public
artifact API to store the artifacts.

