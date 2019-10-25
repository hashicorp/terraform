---
layout: "registry"
page_title: "Terraform Registry - Provider Documentation"
sidebar_current: "docs-registry-provider-docs"
description: |-
  Expected document structure for publishing providers to the Terraform Registry.
---

# Provider Documentation

This describes the expected document structure for publishing providers to the [Terraform Registry][terraform-registry].

## Publishing

~> Publishing is currently in a closed beta. Although we do not expect this document to change significantly before opening provider publishing to the community, this reference currently only applies to providers already appearing on the [Terraform Registry providers list][terraform-registry-providers].

The Terraform Registry publishes providers from their Git repositories, creating a version for each Git tag that matches the [Semver](https://semver.org/) versioning format. Provider documentation is published automatically as part of the provider release process.

Provider documentation is always tied to a provider version. A given version always displays the documentation from that version's Git commit, and the only way to publish updated documentation is to release a new version of the provider.

### Storage Limits

The maximum number of documents allowed for a single provider version is 1000.

Each document can contain no more than 500KB of data. Documents which exceed this limit will be truncated, and a note will be displayed in the Terraform Registry.

## Format

Provider documentation should be a directory of Markdown documents in the provider repository. Each Markdown document is rendered as a separate page. The directory should include a document for the provider index, a document for each Resource and Data Source, and optional documents for any Guides.

### Directory Structure

| Location | Filename | Description |
|-|-|-|
| `docs/` | `index.md` | Index page for the provider. |
| `docs/guides/` | `<guide>.md` | Additional documentation for Guides. |
| `docs/resources/` | `<resource>.md` | Information for a Resource. Filename should not include a `<PROVIDER NAME>_` prefix. |
| `docs/data-sources/` | `<data_source>.md` | Information on a provider Data Source. |

-> **Note:** In order to support provider docs which have already been formatted for publishing to [terraform.io][terraform-io-providers], the Terraform Registry also supports docs in a `website/docs/` legacy directory with file extensions of `.html.markdown` or `.html.md`.

### Headers

We strongly suggest that provider docs include the following sections to help users understand how to use the provider. Create additional sections if they would enhance usability of the Resource (for example, “Imports” or “Customizable Timeouts”).

#### Index

    # <provider> Provider

    Summary of what the provider is for, including use cases and links to app/service documentation.

    ## Example Usage

    ```terraform
    // Code block with an example of how to use this provider.
    ```
    
    ## Argument Reference
    
    * List any arguments for the provider block.

#### Resources/Data Sources

    # <resource name> Resource/Data Source

    Description of what this Resource does, with links to official app/service documentation.
    
    ## Example Usage
    
    ```terraform
    // Code block with an example of how to use this Resource.
    ```
    
    ## Argument Reference
    
    * `attribute_name` - (Optional/Required) List arguments this Resource takes.
    
    ## Attribute Reference
    
    * `attribute_name` - List attributes that this Resource exports.

### YAML Frontmatter

Markdown source files may contain YAML frontmatter, which provides information to the Registry for organization and display of providers.

Frontmatter is not rendered in the Terraform Registry web UI.

#### Example

```markdown
---
page_title: "Authenticating with Foo Service via OAuth"
subcategory: "Authentication"
---
```

#### Supported Attributes

The following frontmatter attributes are supported by the Terraform Registry:

* **page_title** - The title of this document, which will display in the docs navigation. This is only required for documents in the `guides/` folder.
* **subcategory** - An optional additional layer of grouping that affects the display of the docs navigation; [see Subcategories below](#subcategories) for more details. Resources and Data Sources should be organized into subcategories if the number of Resources would be difficult to quickly scan for a user. Guides should be separated into subcategories if there are multiple Guides which fit into 2 or more distinct groupings.

### Callouts

If you start a paragraph with a special arrow-like sigil, it will become a colored callout box. You can't make multi-paragraph callouts. For colorblind users (and for clarity in general), callouts will automatically start with a strong-emphasized word to indicate their function.

Sigil | Text prefix       | Color
------|-------------------|-------
`->`  | `**Note**`       | blue
`~>`  | `**Note**`       | yellow
`!>`  | `**Warning**`    | red

## Navigation Hierarchy

Provider docs are organized by category: Resources, Data Sources, and Guides. At a minimum, a provider must contain an index (`docs/index.md`) and at least one Resource or Data Source.

### Typical Structure

A provider named `example` with a Resource and Data Source for `cloud` would have these 3 files:

```
docs/
    index.md
    data-sources/
        cloud.md
    Resources/
        cloud.md
```

After publishing this provider version, its page on the Terraform Registry would display a navigation which resembles this hierarchy:

* example Provider
* Resources
    * example_cloud
* Data Sources
    * example_cloud

### Subcategories

If we wanted to group these Resources by a service or other dimension, we could add a `subcategory` field to the YAML frontmatter of the Resource and Data Source:

```markdown
---
subcategory: "Cloud"
---
```

This would change the navigation hierarchy to the following:

* example Provider
* Cloud
    * Resources
        * example_cloud
    * Data Sources
        * example_cloud

Resources and Data Sources without a subcategory will be rendered before any subcategories.

### Guides

Providers can optionally include 1 or more Guides. These can assist users in using the provider for certain scenarios.

```
docs/
    index.md
    Guides/
        authenticating.md
    data-sources/
        cloud.md
    Resources/
        cloud.md
```

The title for Guides is controlled with the `page_title` attribute in the YAML frontmatter:

```markdown
---
page_title: "Authenticating with Example Cloud"
---
```

The `page_title` is used (instead of the filename) for rendering the link to this guide in the navigation:

* example Provider
* Guides
    * Authenticating with Example Cloud
* Resources
    * example_cloud
* Data Sources
    * example_cloud

Guides are always rendered before Resources, Data Sources, and any subcategories.

If a `page_title` attribute is not found, the title will default to the filename without the extension.

### Guides Subcategories

If a provider has many Guides, you can use subcategories to group them into separate top-level sections. For example, given the following directory structure:

```
docs/
    index.md
    Guides/
        authenticating-basic.md
        authenticating-oauth.md
        setup.md
    data-sources/
        cloud.md
    Resources/
        cloud.md
```

Assuming that these three Guides have titles similar to their filenames, and the first two include `subcategory: "Authentication"` in their frontmatter, the Terraform Registry would display this navigation structure:

* example Provider
* Guides
    * Initial Setup
* Authentication
    * Authenticating with Basic Authentication
    * Authenticating with OAuth
* Resources
    * example_cloud
* Data Sources
    * example_cloud

Guides without a subcategory are always rendered before Guides with subcategories. Both are always rendered before Resources and Data Sources.

## Migrating Legacy Providers Docs

For most provider docs already published to [terraform.io][terraform-io-providers], no changes are required to publish them to the Terraform Registry.

~> **Important:** The only exceptions are providers which organize Resources, Data Sources, or Guides into subcategories. See the [Subcategories](#subcategories) section above for more information.

If you want to publish docs on the Terraform Registry that are not currently published to terraform.io, take the following steps to migrate to the newer format:

1. Move the `website/docs/` folder to `docs/`
2. Expand the folder names to match the Terraform Registry's expected format:
    * Rename `docs/d/` to `docs/data-sources/`
    * Rename `docs/r/` to `docs/resources/`
3. Change file suffixes from `.html.markdown` or `.html.md` to `.md`.

[terraform-registry]: https://registry.terraform.io
[terraform-registry-providers]: https://registry.terraform.io/browse/providers
[terraform-io-providers]: https://www.terraform.io/docs/providers/
