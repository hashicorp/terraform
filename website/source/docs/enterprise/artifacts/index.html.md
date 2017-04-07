---
layout: "enterprise"
page_title: "Artifacts - Terraform Enterprise"
sidebar_current: "docs-enterprise-artifacts"
description: |-
  Terraform Enterprise can be used to store artifacts for use by Terraform. Typically, artifacts are stored with Packer.
---

# About Terraform Artifacts

Terraform Enterprise can be used to store artifacts for use by Terraform.
Typically, artifacts are [stored with Packer](https://packer.io/docs).

Artifacts can be used in to deploy and manage images
of configuration. Artifacts are generic, but can be of varying types
like `amazon.image`. See the Packer [`artifact_type`](https://packer.io/docs/post-processors/atlas.html#artifact_type)
docs for more information.

Packer can create artifacts both while running in and out of Terraform
Enterprise network. This is possible due to the post-processors use of the
public artifact API to store the artifacts.
