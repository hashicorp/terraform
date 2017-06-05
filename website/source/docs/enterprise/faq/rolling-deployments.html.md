---
layout: "enterprise"
page_title: "Rolling Deployments - FAQ - Terraform Enterprise"
sidebar_current: "docs-enterprise-faq-deployments"
description: |-
  How do I configure rolling deployments in Terraform Enterprise?
---

# Rolling Deployments

*How do I configure rolling deployments?*

User are able to quickly change out an Artifact version that is being utilized
by Terraform, using variables within Terraform Enterprise. This is particularly
useful when testing specific versions of the given artifact without performing a
full rollout. This configuration also allows one to deploy any version of an
artifact with ease, simply by changing a version variable in Terraform and
re-deploying.

Here is an example:

```hcl
variable "type"           { default = "amazon.image" }
variable "region"         {}
variable "atlas_username" {}
variable "pinned_name"    {}
variable "pinned_version" { default = "latest" }

data "atlas_artifact" "pinned" {
  name     = "${var.atlas_username}/${var.pinned_name}"
  type     = "${var.type}"
  version  = "${var.pinned_version}"

  lifecycle { create_before_destroy = true }

  metadata {
    region = "${var.region}"
  }
}

output "pinned" { value = "${atlas_artifact.pinned.metadata_full.ami_id}" }
```


In the above example we have an `atlas_artifact` resource where you pass in the
version number via the variable `pinned_version`. (_note: this variable defaults
to latest_). If you ever want to deploy any other version, you just update the
variable `pinned_version` and redeploy.

Below is similar to the first example, but it is in the form of a module that
handles the creation of artifacts:

```hcl
variable "type"             { default = "amazon.image" }
variable "region"           {}
variable "atlas_username"   {}
variable "artifact_name"    {}
variable "artifact_version" { default = "latest" }

data "atlas_artifact" "artifact" {
  name    = "${var.atlas_username}/${var.artifact_name}"
  type    = "${var.type}"
  count   = "${length(split(",", var.artifact_version))}"
  version = "${element(split(",", var.artifact_version), count.index)}"

  lifecycle { create_before_destroy = true }
  metadata  { region = "${var.region}" }
}

output "amis" { value = "${join(",", atlas_artifact.artifact.*.metadata_full.ami_id)}" }
```

One can then use the module as follows (_note: the source will likely be
different depending on the location of the module_):

```hcl
module "artifact_consul" {
  source = "../../../modules/aws/util/artifact"

  type             = "${var.artifact_type}"
  region           = "${var.region}"
  atlas_username   = "${var.atlas_username}"
  artifact_name    = "${var.consul_artifact_name}"
  artifact_version = "${var.consul_artifacts}"
}
```


In the above example, we have created artifacts for Consul. In this example, we
can create two versions of the artifact, "latest" and "pinned". This is useful
when rolling a cluster (like Consul) one node at a time, keeping some nodes
pinned to current version and others deployed with the latest Artifact.

There are additional details for implementing rolling deployments in the [Best-Practices Repo](https://github.com/hashicorp/best-practices/blob/master/terraform/providers/aws/us_east_1_prod/us_east_1_prod.tf#L105-L123), as there are some things uncovered in this FAQ (i.e Using the Terraform Enterprise Artifact in an instance).
