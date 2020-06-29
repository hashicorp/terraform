---
layout: "registry"
page_title: "Terraform Registry - Publishing Providers"
sidebar_current: "docs-registry-provider-publishing"
description: |-
  Publishing Providers to the Terraform Registry
---

-> __Publishing Beta__<br>Welcome! Thanks for your interest participating in our Providers in the Registry beta! Paired with Terraform 0.13, our vision is to make it easier than ever to discover, distribute, and maintain your provider(s). We welcome any feedback you have throughout the process and encourage you to reach out if you have any questions or issues by emailing terraform-registry-beta@hashicorp.com.

## Preparing your Provider

### Writing a Provider

Providers published to the Terraform Registry are written and built in the same way as other Terraform Providers. For guidance on how to write a provider, see [Writing Custom Providers](/docs/extend/writing-custom-providers.html).

The provider repository on GitHub must match the pattern `terraform-provider-{NAME}`, and the repository must be public.

### Documenting your Provider

Your provider should contain an overview document (index.md), as well as a doc for each resource and data-source. See [Documenting Providers](documenting.html) for details about how to ensure your provider documentation renders properly on the Terraform Registry.

### Creating a GitHub Release

Publishing a provider requires at least one version be available on GitHub Releases. The tag must be a valid [Semantic Version](https://semver.org/) preceded with a `v` (for example, `v1.2.3`).

Terraform CLI and the Terraform Registry follow the Semantic Versioning specification when detecting a valid version, sorting versions, solving version constraints, and choosing the latest version. Prerelease versions are supported (available if explicitly defined but not chosen automatically) with a hyphen (-) delimiter, such as `v1.2.3-pre`.

#### Using GoReleaser locally

GoReleaser is a tool for building Go projects for multiple platforms, creating a checksums file, and signing the release. It can also upload your release to GitHub Releases.

1. Install [GoReleaser](https://goreleaser.com) using the [installation instructions](https://goreleaser.com/install/).
1. Copy the [.goreleaser.yml file](https://github.com/hashicorp/terraform-provider-scaffolding/blob/master/.goreleaser.yml) from the hashicorp/scaffolding provider repository.
1. Cache the password for your GPG private key with `gpg --armor --detach-sign` (see note below).
1. Set your `GITHUB_TOKEN` to a [Personal Access Token](https://github.com/settings/tokens) that has the **public_repo** scope.
1. Tag your version with `git tag v1.2.3`.
1. Build, sign, and upload your release with `goreleaser release --rm-dist`.

-> GoReleaser does not support signing binaries with a GPG key that requires a passphrase. Some systems may cache your GPG passphrase for a few minutes. If you are unable to cache the passphrase for GoReleaser, please use the manual release preparation process below, or remove the signature step from GoReleaser and sign it prior to moving the GitHub release from draft to published.

#### Manually Preparing a Release

If for some reason you're not able to use GoReleaser to build, sign, and upload your release, you can create the required assets by following these steps, or encode them into a Makefile or shell script.

The release must meet the following criteria:

* There are 1 or more zip files containing the built provider binary for a single architecture
    * The binary name is `terraform-provider-{NAME}_v{VERSION}`
    * The archive name is `terraform-provider-{NAME}_{VERSION}_{OS}_{ARCH}.zip`
* There is a `terraform-provider-{NAME}_{VERSION}_SHA256SUMS` file, which contains a sha256 sum for each zip file in the release.
    * `shasum -a 256 *.zip > terraform-provider-{NAME}_{VERSION}_SHA256SUMS`
* There is a `terraform-provider-{NAME}_{VERSION}_SHA256SUMS.sig` file, which is a valid GPG signature of the `terraform-provider-{NAME}_{VERSION}_SHA256SUMS` file using the keypair.
    * `gpg --detach-sign terraform-provider-{NAME}_{VERSION}_SHA256SUMS`
* Release is finalized (not a private draft).

## Publishing to the Registry

### Creating a Terraform Registry Account

Before publishing a provider, you must first authenticate to the Terraform Registry with a GitHub account. The account must have admin permissions on the provider repository to create the required webhooks for publishing future provider versions.

Click [Sign-In](https://registry.terraform.io/sign-in) to authenticate to the Terraform Registry with your GitHub user account.

### Adding Your GPG Signing Key

All provider releases are required to be signed, thus you must provide HashiCorp with the public key for the GPG keypair that you will be signing releases with. The Terraform Registry will validate that the release is signed with this key when publishing each version, and Terraform will verify this during `terraform init`.

To export your public key in ASCII-armor format, use the following command:

```console
$ gpg --armor --export "{Key ID or email address}"
```

#### Individuals

If you would like to publish a provider under your username (not a GitHub organization), you can add your GPG key to the Terraform Registry by visiting [User Settings > Signing Keys](https://registry.terraform.io/settings/gpg-keys).

#### Organizations

In order to publish a provider under a GitHub organization, your public key must be added to the Terraform Registry by a HashiCorp employee. You can email it to terraform-registry@hashicorp.com, or your HashiCorp contact person (if you have one).

### Publishing Your provider

In the top-right navigation, select [Publish > Provider](https://registry.terraform.io/publish/provider) to begin the publishing process. Follow the prompts to select the organization and repository you would like to publish.
