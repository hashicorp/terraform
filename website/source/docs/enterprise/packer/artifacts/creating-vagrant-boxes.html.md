---
layout: "enterprise"
page_title: "Creating Vagrant Boxes - Packer Artifacts - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerartifacts-vagrant"
description: |-
  Creating Vagrant artifacts with Terraform Enterprise.
---

# Creating Vagrant Boxes with Packer

We recommend using Packer to create boxes, as is it is fully repeatable and
keeps a strong history of changes within Terraform Enterprise.

## Getting Started

Using Packer requires more up front effort, but the repeatable and automated
builds will end any manual management of boxes. Additionally, all boxes will be
stored and served from Terraform Enterprise, keeping a history along the way.

## Post-Processors

Packer uses
[post-processors](https://packer.io/docs/templates/post-processors.html) to
define how to process images and artifacts after provisioning. Both the
`vagrant` and `atlas` post-processors must be used in order to upload Vagrant
Boxes to Terraform Enterprise via Packer.

It's important that they are [sequenced](https://packer.io/docs/templates/post-processors.html)
in the Packer template so they run in order. This is done by nesting arrays:

```javascript
{
  "post-processors": [
    [
      {
        "type": "vagrant"
        // ...
      },
      {
        "type": "atlas"
        // ...
      }
    ]
  ]
}
```

Sequencing automatically passes the resulting artifact from one
post-processor to the next – in this case, the `.box` file.

### Vagrant Post-Processor

The [Vagrant post-processor](https://packer.io/docs/post-processors/vagrant.html) is required to package the image
from the build (an `.ovf` file, for example) into a `.box` file before
passing it to the `atlas` post-processor.

```json
{
  "type": "vagrant",
  "keep_input_artifact": false
}
```

The input artifact (i.e and `.ovf` file) does not need to be kept when building Vagrant Boxes,
as the resulting `.box` will contain it.

### Post-Processor

The [post-processor](https://packer.io/docs/post-processors/atlas.html) takes the resulting `.box` file and uploads it adding metadata about the box version.

```json
{
  "type": "atlas",
  "artifact": "my-username/dev-environment",
  "artifact_type": "vagrant.box",
  "metadata": {
    "provider": "vmware_desktop",
    "version": "0.0.1"
  }
}
```

#### Attributes Required

These are all of the attributes for that post-processor
required for uploading Vagrant Boxes. A complete example is shown below.

- `artifact`: The username and box name (`username/name`) you're creating the version
of the box under. If the box doesn't exist, it will be automatically
created
- `artifact_type`: This must be `vagrant.box`. Terraform Enterprise uses this to determine
how to treat this artifact.

For `vagrant.box` type artifacts, you can specify keys in the metadata block:

- `provider`: The Vagrant provider for the box. Common providers are
`virtualbox`, `vmware_desktop`, `aws` and so on _(required)_
- `version`: This is the Vagrant box version and is constrained to the
same formatting as in the web UI: `*.*.*` _(optional, but required for boxes
with multiple providers). The version will increment on the minor version if left blank (e.g the initial version will be set to 0.1.0, the subsequent version will be set to 0.2.0)._
- `description`: This is the description that will be shown with the
version of the box. You can use Markdown for links and style. _(optional)_

## Example

An example post-processor block for Terraform Enterprise and Vagrant is below. In this example,
the build runs on both VMware and Virtualbox creating two
different providers for the same box version (`0.0.1`).

```json
{
  "post-processors": [
    [
      {
        "type": "vagrant",
        "keep_input_artifact": false
      },
      {
        "type": "atlas",
        "only": ["vmware-iso"],
        "artifact": "my-username/dev-environment",
        "artifact_type": "vagrant.box",
        "metadata": {
          "provider": "vmware_desktop",
          "version": "0.0.1"
        }
      },
      {
        "type": "atlas",
        "only": ["virtualbox-iso"],
        "artifact": "my-username/dev-environment",
        "artifact_type": "vagrant.box",
        "metadata": {
          "provider": "virtualbox",
          "version": "0.0.1"
        }
      }
    ]
  ]
}
```
