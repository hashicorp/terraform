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

### Versioning

Docs are imported to the Terraform Registry for each Git tag matching the Semver versioning format. Updates will not be visible in the Terraform Registry until a new provider version is released.

### Storage Limits

The maximum number of documents for a single provider version allowed is 1000.

Each document can contain no more than 500KB of data. Documents which exceed this limit will be truncated, and a note will be displayed in the Terraform Registry.

## Format

Provider documentation is expected to be a folder in the provider repo, containing a Markdown document for the provider overview, and for each resource, data source, and (optionally) guides.

### Folder Structure

| Location | Filename | Description |
|-|-|-|
| `docs/` | `index.md` | Overview page for the provider|
| `docs/guides/` | `<guide>.md` | Additional documentation for guides |
| `docs/resources/` | `<resource>.md` | Information on a provider resource |
| `docs/data-sources/` | `<data_source>.md` | Information on a provider data source |

-> In order to support provider docs which have already been formatted for publishing to [terraform.io](terraform-io-providers), the Terraform Registry also supports docs in a `website/docs/` legacy directory with file extensions of `.html.markdown` or `.html.md`.

### Headers

We strongly suggest that provider docs include the following sections to help users understand how to use the provider. Additional sections should also be created if they would enhance usability of the resource (for example “Imports” or “Customizable Timeouts”).

#### Overview

    # <provider> Provider

    Summary of what the providers is for, including use cases and links to app/service documentation

    ## Example Usage

    ```terraform
    // Code block with an example of how to use this provider.
    ```
    
    ## Argument Reference
    
    * List any arguments for the provider block.

#### Resources/Data Sources

    # <resource name> Resource/Data Source

    Description of what this resource does, with links to official app/service documentation.
    
    ## Example Usage
    
    ```terraform
    // Code block with an example of how to use this resource.
    ```
    
    ## Argument Reference
    
    * List arguments this resource takes.
    
    ## Attribute Reference
    
    * List attributes that this resource exports.

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
* **subcategory** - An optional additional layer of grouping for the docs navigation. Resources and Data Sources should be organized into subcategories if the number of resources would be difficult to quickly scan for a user. Guides should be separated into subcategories if there are multiple guides which fit into 2 or more distinct groupings.

### Callouts

If you start a paragraph with a special arrow-like sigil, it will become a colored callout box. You can't make multi-paragraph callouts. For colorblind users (and for clarity in general), we try to start callouts with a strong-emphasized word to indicate their function.

Sigil | Start text with   | Color
------|-------------------|-------
`->`  | `**Note:**`       | blue
`~>`  | `**Important:**`  | yellow
`!>`  | `**Warning:**`    | red

## Navigation Hierarchy

Provider docs will be organized by categories, which are derived from expected subdirectory names. At a minimum, a provider must contain an Overview (`docs/index.md`, and at least one resource or data source.

### Typical Structure

A provider named `example` with a resource and data source for `cloud` would have these 3 files:

```
docs/
    index.md
    data-sources/
        cloud.md
    resources/
        cloud.md
```

After publishing this provider version, viewing the documentation for this provider on the Terraform Registry would display a navigation which resembles this hierarchy:

* example Provider
* Resources
    * example_cloud
* Data Sources
    * example_cloud

### Subcategories

If we wanted to group these resources by a service or other dimension, we could add a `subcategory` field to the YAML frontmatter of the resource and data source:

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

Resources and data sources without a subcategory will be rendered before any subcategories.

### Guides

Providers can optionally include 1 or more guides. These can assist users in using the provider for certain scenarios.

```
docs/
    index.md
    guides/
        authenticating.md
    data-sources/
        cloud.md
    resources/
        cloud.md
```

The title for guides is controlled with the `page_title` attribute in the YAML frontmatter:

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

### Guides Subcategories

If a provider has many guides, it can be useful to group them into separate top-level folders. These also use the `subcategory` attribute in the YAML frontmatter:

```
docs/
    index.md
    guides/
        authenticating-basic.md
        authenticating-oauth.md
        setup.md
    data-sources/
        cloud.md
    resources/
        cloud.md
```

Given these 3 guides have titles similar to their filename, and `subcategory: "Authentication"` has been added for the first two, the Terraform Registry would display this navigation structure:

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

Guides without a subcategory are always rendered before guides with subcategories. Both are always rendered before Resources and Data Sources.

## Migrating Legacy Providers Docs

For most provider docs already published to [terraform.io][terraform-io-providers], no changes are required to publish them to the Terraform Registry.

~> The only exception is any providers which organize resources, data sources, or guides into subcategories. See the [Subcategories](#subcategories) section above for more information.

For provider docs which are not being published to terraform.io, but wish to be published to the Terraform Registry, take the following steps to migrate to the newer format:

1. Move the `website/docs/` folder to `docs/`
2. Expand the folder names to be more explicit:
    * Rename `docs/d/` to `docs/data-sources/`
    * Rename `docs/r/` to `docs/resources/`
3. Change file suffixes from `.html.markdown` or `.html.md` to `.md`.

[terraform-registry]: https://registry.terraform.io
[terraform-registry-providers]: https://registry.terraform.io/browse/providers
[terraform-io-providers]: https://www.terraform.io/docs/providers/
