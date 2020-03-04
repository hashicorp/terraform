---
layout: "registry"
page_title: "Terraform Registry - Provider Documentation"
sidebar_current: "docs-registry-provider-docs"
description: |-
  Expected document structure for publishing providers to the Terraform Registry.
---

# Provider Documentation

The [Terraform Registry][terraform-registry] displays documentation for the providers it hosts. This page describes the expected format for provider documentation.

## Publishing

-> **Note:** Publishing is currently in a closed beta. Although we do not expect this document to change significantly before opening provider publishing to the community, this reference currently only applies to providers already appearing on the [Terraform Registry providers list][terraform-registry-providers].

The Terraform Registry publishes providers from their Git repositories, creating a version for each Git tag that matches the [Semver](https://semver.org/) versioning format. Provider documentation is published automatically as part of the provider release process.

Provider documentation is always tied to a provider version. A given version always displays the documentation from that version's Git commit, and the only way to publish updated documentation is to release a new version of the provider.

### Storage Limits

The maximum number of documents allowed for a single provider version is 1000.

Each document can contain no more than 500KB of data. Documents which exceed this limit will be truncated, and a note will be displayed in the Terraform Registry.

## Format

Provider documentation should be a directory of Markdown documents in the provider repository. Each Markdown document is rendered as a separate page. The directory should include a document for the provider index, a document for each resource and data source, and optional documents for any guides.

### Directory Structure

| Location | Filename | Description |
|-|-|-|
| `docs/` | `index.md` | Index page for the provider. |
| `docs/guides/` | `<guide>.md` | Additional documentation for guides. |
| `docs/resources/` | `<resource>.md` | Information for a Resource. Filename should not include a `<PROVIDER NAME>_` prefix. |
| `docs/data-sources/` | `<data_source>.md` | Information on a provider data source. |

-> **Note:** In order to support provider docs which have already been formatted for publishing to [terraform.io][terraform-io-providers], the Terraform Registry also supports docs in a `website/docs/` legacy directory with file extensions of `.html.markdown` or `.html.md`.

### Headers

We strongly suggest that provider docs include the following sections to help users understand how to use the provider. Create additional sections if they would enhance usability of the resource (for example, “Imports” or “Customizable Timeouts”).

#### Index Headers

    # <provider> Provider

    Summary of what the provider is for, including use cases and links to
    app/service documentation.

    ## Example Usage

    ```hcl
    // Code block with an example of how to use this provider.
    ```

    ## Argument Reference

    * List any arguments for the provider block.

#### Resource/Data Source Headers

    # <resource name> Resource/Data Source

    Description of what this resource does, with links to official
    app/service documentation.

    ## Example Usage

    ```hcl
    // Code block with an example of how to use this resource.
    ```

    ## Argument Reference

    * `attribute_name` - (Optional/Required) List arguments this resource takes.

    ## Attribute Reference

    * `attribute_name` - List attributes that this resource exports.

### YAML Frontmatter

Markdown source files may contain YAML frontmatter, which provides organizational information and display hints. Frontmatter can be omitted for resources and data sources that don't require a subcategory.

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
* **subcategory** - An optional additional layer of grouping that affects the display of the docs navigation; [see Subcategories below](#subcategories) for more details. Resources and data sources should be organized into subcategories if the number of resources would be difficult to quickly scan for a user. Guides should be separated into subcategories if there are multiple guides which fit into 2 or more distinct groupings.

### Callouts

If you start a paragraph with a special arrow-like sigil, it will become a colored callout box. You can't make multi-paragraph callouts. For colorblind users (and for clarity in general), callouts will automatically start with a strong-emphasized word to indicate their function.

Sigil | Text prefix       | Color
------|-------------------|-------
`->`  | `**Note**`       | blue
`~>`  | `**Note**`       | yellow
`!>`  | `**Warning**`    | red

## Navigation Hierarchy

Provider docs are organized by category: resources, data sources, and guides. At a minimum, a provider must contain an index (`docs/index.md`) and at least one resource or data source.

### Typical Structure

A provider named `example` with a resource and data source for `instance` would have these 3 files:

```
docs/
    index.md
    data-sources/
        instance.md
    resources/
        instance.md
```

After publishing this provider version, its page on the Terraform Registry would display a navigation which resembles this hierarchy:

* example Provider
* Resources
    * example_instance
* Data Sources
    * example_instance

### Subcategories

To group these resources by a service or other dimension, add the optional `subcategory` field to the YAML frontmatter of the resource and data source:

```markdown
---
subcategory: "Compute"
---
```

This would change the navigation hierarchy to the following:

* example Provider
* Compute
    * Resources
        * example_instance
    * Data Sources
        * example_instance

Resources and data sources without a subcategory will be rendered before any subcategories.

### Guides

Providers can optionally include 1 or more guides. These can assist users in using the provider for certain scenarios.

```
docs/
    index.md
    guides/
        authenticating.md
    data-sources/
        instance.md
    resources/
        instance.md
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
    * example_instance
* Data Sources
    * example_instance

Guides are always rendered before resources, data sources, and any subcategories.

If a `page_title` attribute is not found, the title will default to the filename without the extension.

### Guides Subcategories

If a provider has many guides, you can use subcategories to group them into separate top-level sections. For example, given the following directory structure:

```
docs/
    index.md
    guides/
        authenticating-basic.md
        authenticating-oauth.md
        setup.md
    data-sources/
        instance.md
    resources/
        instance.md
```

Assuming that these three guides have titles similar to their filenames, and the first two include `subcategory: "Authentication"` in their frontmatter, the Terraform Registry would display this navigation structure:

* example Provider
* Guides
    * Initial Setup
* Authentication
    * Authenticating with Basic Authentication
    * Authenticating with OAuth
* Resources
    * example_instance
* Data Sources
    * example_instance

Guides without a subcategory are always rendered before guides with subcategories. Both are always rendered before resources and data sources.

## Migrating Legacy Providers Docs

For most provider docs already published to [terraform.io][terraform-io-providers], no changes are required to publish them to the Terraform Registry.

~> **Important:** The only exceptions are providers which organize resources, data sources, or guides into subcategories. See the [Subcategories](#subcategories) section above for more information.

If you want to publish docs on the Terraform Registry that are not currently published to terraform.io, take the following steps to migrate to the newer format:

1. Move the `website/docs/` folder to `docs/`
2. Expand the folder names to match the Terraform Registry's expected format:
    * Rename `docs/d/` to `docs/data-sources/`
    * Rename `docs/r/` to `docs/resources/`
3. Change file suffixes from `.html.markdown` or `.html.md` to `.md`.

[terraform-registry]: https://registry.terraform.io
[terraform-registry-providers]: https://registry.terraform.io/browse/providers
[terraform-io-providers]: https://www.terraform.io/docs/providers/
