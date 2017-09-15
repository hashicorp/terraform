---
layout: "registry"
page_title: "Terraform Registry - Publishing Modules"
sidebar_current: "docs-registry-publish"
description: |-
  Anyone can publish and share modules on the Terraform Registry.
---

# Publishing Modules

Anyone can publish and share modules on the [Terraform Registry](https://registry.terraform.io).

Published modules support versioning, automatically generate documentation,
allow browsing version histories, show examples and READMEs, and more. We
recommend publishing reusable modules to a registry.

Public modules are managed via Git and GitHub. Publishing a module takes only
a few minutes. Once a module is published, you can release a new version of
a module by simply pushing a properly formed Git tag.

The registry extracts the name of the module, the provider, the documentation,
inputs/outputs, and more directly from the source of the module. No manual
annotations are required.

## Requirements

The list below contains all the requirements for publishing a module.
Meeting the requirements for publishing a module is extremely easy. The
list may appear long only to ensure we're detailed, but adhering to the
requirements should happen naturally.

* **GitHub.** The module must be on GitHub and must be a public repo.
This is only a requirement for the [public registry](https://registry.terraform.io).
If you're using a private registry, you may ignore this requirement.

* **Repository name.** The repository name must be `terraform-PROVIDER-NAME`
where PROVIDER is the primary provider to associate with the module and
NAME is a unique name for the module. The name may contain hyphens. Example:
`terraform-aws-consul` or `terraform-google-vault`.

* **Repository description.** The GitHub repository description is used
to populate the short description of the module. This should be a simple
one sentence description of the module.

* **Standard Module Structure.** The module must adhere to the
[standard module structure](/docs/modules/create.html#standard-module-structure).
This allows the registry to inspect your module and generate documentation,
track resource usage, and more.

* **Tags for Releases.** Releases are detected by creating and pushing
tags. The tag name must be a semantic version that can optionally be prefixed
with a `v`. Examples are `v1.0.4` and `0.9.2`. To publish a module initially,
at least one release tag must be present.

## Publishing a Public Module

With the requirements met, you can publish a public module by going to
the [Terraform Registry](https://registry.terraform.io) and clicking the
"Upload" link in the top navigation.

If you're not signed in, this will ask you to connect with GitHub. We only
ask for access to public repositories, since the public registry may only
publish public modules. We require access to hooks so we can register a webhook
with your repository. We require access to your email address so that we can
email you alerts about your module. We will not spam you.

The upload page will list your available repositories, filtered to those that
match the [naming convention described above](#Requirements). This is shown in
the screenshot below. Select the repository of the module you want to add and
click "Create Module."

In a few seconds, your module will be created.

![Create Module flow animation](/assets/images/docs/registry-upload.gif)

## Releasing New Versions

The Terraform Registry uses tags to detect releases.

Tag names must be a valid [semantic version](http://semver.org), optionally
prefixed with a `v`. Example of valid tags are: `v1.0.1` and `0.9.4`. To publish
a new module, you must already have at least one tag created.

To release a new version, create and push a new tag with the proper format.
The webhook will notify the registry of the new version and it will appear
on the registry usually in less than a minute.

If your version doesn't appear properly, you may force a sync with GitHub
by viewing your module on the registry and clicking "Force GitHub Sync"
under the "Manage Module" dropdown. This process may take a few minutes.
Please only do this if you do not see the version appear, since it will
cause the registry to resync _all versions_ of your module.
