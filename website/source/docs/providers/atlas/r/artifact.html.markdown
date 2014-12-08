---
layout: "atlas"
page_title: "Atlas: atlas_artifact"
sidebar_current: "docs-atlas-resource-artifact"
description: |-
  Provides access to deployment artifacts managed by Atlas. This can
  be used to dynamically configure instantiation and provisioning
  of resources.
---

# atlas\_artifact

Provides access to deployment artifacts managed by Atlas. This can
be used to dynamically configure instantiation and provisioning
of resources.

## Example Usage

```
# Read the AMI
resource "atlas_artifact" "web" {
    name = "hashicorp/web"
    type = "aws.ami"
    metadata {
        arch = "386"
    }
}

# Start our instance with the dynamic ami value
resource "aws_instance" "app" {
    ami = "${atlas_artifact.web.id}"
    ...
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the artifact in Atlas. This is given
  in slug format like "organization/artifact".

* `type` - (Required) The type of artifact to query for.

* `version` - (Optional) By default, if no version is provided the
  latest version of the artifact is used. Providing a version can
  be used to pin a dependency.

* `metadata_keys` - (Optional) If given, only an artifact containing
  the given keys will be returned. This is used to disambiguate when
  multiple potential artifacts match. An example is "aws" to filter
  on an AMI.

* `metadata` - (Optional) If given, only an artifact matching the
  metadata filters will be returned. This is used to disambiguate when
  multiple potential artifacts match. An example is "arch" = "386" to
  filter on architecture.


## Attributes Reference

The following attributes are exported:

* `version` - The matching version of the artifact
* `id` - The ID of the artifact. This could be an AMI ID, GCE Image ID, etc.
* `file_url` - For artifacts that are binaries, this is a download path.
* `metadata_full` - Contains the full metadata of the artifact.

