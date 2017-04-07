---
layout: "enterprise"
page_title: "Build Environment - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-environment"
description: |-
  This page outlines the environment that Packer runs in within Terraform Enterprise.
---

# Packer Build Environment

This page outlines the environment that Packer runs in within Terraform
Enterprise.

### Supported Builders

Terraform Enterprise currently supports running the following Packer builders:

- amazon-chroot
- amazon-ebs
- amazon-instance
- digitalocean
- docker
- googlecompute
- null
- openstack
- qemu
- virtualbox-iso
- vmware-iso

### Files

All files in the uploading package (via [Packer push or GitHub](/docs/enterprise/packer/builds/starting.html)),
and the application from the build pipeline are available on the filesystem
of the build environment.

You can use the file icon on the running build to show a list of
available files.

Files can be copied to the destination image Packer is provisioning
with [Packer Provisioners](https://packer.io/docs/templates/provisioners.html).

An example of this with the Shell provisioner is below.

```json
{
  "provisioners": [
    {
      "type": "shell",
      "scripts": [
        "scripts/vagrant.sh",
        "scripts/dependencies.sh",
        "scripts/cleanup.sh"
      ]
    }
  ]
}
```

We encourage use of relative paths over absolute paths to maintain portability
between Terraform Enterprise and local builds.

The total size of all files in the package being uploaded via
[Packer push or GitHub](/docs/enterprise/packer/builds/starting.html) must be 5 GB or less.

If you need to upload objects that are larger, such as dmgs, see the
[`packer push` "Limits" documentation](https://packer.io/docs/command-line/push.html)
for ways around this limitation.

### Hardware Limitations

Currently, each builder defined in the Packer template receives
the following hardware resources. This is subject to change.

- 1 CPU core
- 2 GB of memory
- 20 GBs of disk space

### Environment Variables

You can set any number of environment variables that will be injected
into your build environment at runtime. These variables can be
used to configure your build with secrets or other key value configuration.

Variables are encrypted and stored securely.

Additionally, the following environment variables are automatically injected. All injected environment variables will be prefixed with `ATLAS_`

- `ATLAS_TOKEN` - This is a unique, per-build token that expires at the end of
  build execution (e.g. `"abcd.atlasv1.ghjkl..."`)
- `ATLAS_BUILD_ID` - This is a unique identifier for this build (e.g. `"33"`)
- `ATLAS_BUILD_NUMBER` - This is a unique identifier for all builds in the same
  scope (e.g. `"12"`)
- `ATLAS_BUILD_NAME` - This is the name of the build (e.g. `"mybuild"`).
- `ATLAS_BUILD_SLUG` - This is the full name of the build
  (e.g. `"company/mybuild"`).
- `ATLAS_BUILD_USERNAME` - This is the username associated with the build
  (e.g. `"sammy"`)
- `ATLAS_BUILD_CONFIGURATION_VERSION` - This is the unique, auto-incrementing
  version for the [Packer build configuration](/docs/enterprise/glossary/index.html) (e.g. `"34"`).
- `ATLAS_BUILD_GITHUB_BRANCH` - This is the name of the branch
  that the associated Packer build configuration version was ingressed from
  (e.g. `master`).
- `ATLAS_BUILD_GITHUB_COMMIT_SHA` - This is the full commit hash
  of the commit that the associated Packer build configuration version was
  ingressed from (e.g. `"abcd1234..."`).
- `ATLAS_BUILD_GITHUB_TAG` - This is the name of the tag
  that the associated Packer build configuration version was ingressed from
  (e.g. `"v0.1.0"`).

If the build was triggered by a new application version, the following
environment variables are also available:

- `ATLAS_APPLICATION_NAME` - This is the name of the application connected to
  the Packer build (e.g. `"myapp"`).
- `ATLAS_APPLICATION_SLUG` - This is the full name of the application connected
  to the Packer build (e.g. `"company/myapp"`).
- `ATLAS_APPLICATION_USERNAME` - This is the username associated with the
  application connected to the Packer build (e.g. `"sammy"`)
- `ATLAS_APPLICATION_VERSION` - This is the version of the application connected
  to the Packer build (e.g. `"2"`).
- `ATLAS_APPLICATION_GITHUB_BRANCH` - This is the name of the branch that the
  associated application version was ingressed from (e.g. `master`).
- `ATLAS_APPLICATION_GITHUB_COMMIT_SHA` - This is the full commit hash
  of the commit that the associated application version was ingressed from
  (e.g. `"abcd1234..."`).
- `ATLAS_APPLICATION_GITHUB_TAG` - This is the name of the tag that the
  associated application version was ingressed from (e.g. `"v0.1.0"`).

For any of the `GITHUB_` attributes, the value of the environment variable will
be the empty string (`""`) if the resource is not connected to GitHub or if the
resource was created outside of GitHub (like using `packer push` or
`vagrant push`).


### Base Artifact Variable Injection

A base artifact can be selected on the "Settings" page for a build
configuration. During each build, the latest artifact version will have it's
external ID (such as an AMI for AWS) injected as an environment variable for the
environment.

The keys for the following artifact types will be injected:

- `aws.ami`: `ATLAS_BASE_ARTIFACT_AWS_AMI_ID`
- `amazon.ami`: `ATLAS_BASE_ARTIFACT_AMAZON_AMI_ID`
- `amazon.image`: `ATLAS_BASE_ARTIFACT_AMAZON_IMAGE_ID`
- `google.image`: `ATLAS_BASE_ARTIFACT_GOOGLE_IMAGE_ID`

You can then reference this artifact in your Packer template, like this
AWS example:

```json
{
  "variables": {
      "base_ami": "{{env `ATLAS_BASE_ARTIFACT_AWS_AMI_ID`}}"
  },
  "builders": [
    {
      "type": "amazon-ebs",
      "access_key": "",
      "secret_key": "",
      "region": "us-east-1",
      "source_ami": "{{user `base_ami`}}"
    }
  ]
}
```

## Notes on Security

Packer environment variables in Terraform Enterprise are encrypted using [Vault](https://vaultproject.io)
and closely guarded and audited. If you have questions or concerns
about the safety of your configuration, please contact our security team
at [security@hashicorp.com](mailto:security@hashicorp.com).
