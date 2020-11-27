---
layout: "language"
page_title: "Providers - Configuration Language"
---

# Providers

Terraform relies on plugins called "providers" to interact with remote systems.

Terraform configurations must declare which providers they require so that
Terraform can install and use them. Additionally, some providers require
configuration (like endpoint URLs or cloud regions) before they can be used.

## What Providers Do

Each provider adds a set of [resource types](/docs/configuration/blocks/resources/index.html)
and/or [data sources](/docs/configuration/data-sources.html) that Terraform can
manage.

Every resource type is implemented by a provider; without providers, Terraform
can't manage any kind of infrastructure.

Most providers configure a specific infrastructure platform (either cloud or
self-hosted). Providers can also offer local utilities for tasks like
generating random numbers for unique resource names.

## Where Providers Come From

Providers are distributed separately from Terraform itself, and each provider
has its own release cadence and version numbers.

The [Terraform Registry](https://registry.terraform.io/browse/providers)
is the main directory of publicly available Terraform providers, and hosts
providers for most major infrastructure platforms.

## How to Use Providers

To use resources from a given provider, you need to include some information
about it in your configuration. See the following pages for details:

- [Provider Requirements](/docs/configuration/provider-requirements.html)
  documents how to declare providers so Terraform can install them.

- [Provider Configuration](/docs/configuration/providers.html)
  documents how to configure settings for providers.

- [Dependency Lock File](/docs/configuration/dependency-lock.html)
  documents an additional HCL file that can be included with a configuration,
  which tells Terraform to always use a specific set of provider versions.

## Provider Installation

- Terraform Cloud and Terraform Enterprise install providers as part of every run.

- Terraform CLI finds and installs providers when
  [initializing a working directory](/docs/cli/init/index.html). It can
  automatically download providers from a Terraform registry, or load them from
  a local mirror or cache. If you are using a persistent working directory, you
  must reinitialize whenever you change a configuration's providers.

    To save time and bandwidth, Terraform CLI supports an optional plugin
    cache. You can enable the cache using the `plugin_cache_dir` setting in
    [the CLI configuration file](/docs/commands/cli-config.html).

To ensure Terraform always installs the same provider versions for a given
configuration, you can use Terraform CLI to create a
[dependency lock file](/docs/configuration/dependency-lock.html)
and commit it to version control along with your configuration. If a lock file
is present, Terraform Cloud, CLI, and Enterprise will all obey it when
installing providers.

## How to Find Providers

To find providers for the infrastructure platforms you use, browse
[the providers section of the Terraform Registry](https://registry.terraform.io/browse/providers).

Some providers on the Registry are developed and published by HashiCorp, some
are published by platform maintainers, and some are published by users and
volunteers. The provider listings use the following badges to indicate who
develops and maintains a given provider.

<table border="0" style="border-collapse: collapse; width: 100%;">
<tbody>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><strong>Tier</strong></td>
<td style="width: 55.7271%; height: 21px;"><strong>Description</strong></td>
<td style="width: 31.7889%; height: 21px;"><strong>Namespace</strong></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="/docs/registry/providers/images/official-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;"><i><span style="font-weight: 400;">Official providers are owned and maintained by HashiCorp </span></i></td>
<td style="width: 31.7889%; height: 21px;"><code><span style="font-weight: 400;">hashicorp</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="/docs/registry/providers/images/verified-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;"><i><span style="font-weight: 400;">Verified providers are owned and maintained by third-party technology partners. Providers in this tier indicate HashiCorp has verified the authenticity of the Provider&rsquo;s publisher, and that the partner is a member of the </span></i><a href="https://www.hashicorp.com/ecosystem/become-a-partner/"><i><span style="font-weight: 400;">HashiCorp Technology Partner Program</span></i></a><i><span style="font-weight: 400;">.</span></i></td>
<td style="width: 31.7889%; height: 21px;"><span style="font-weight: 400;">Third-party organization, e.g. </span><code><span style="font-weight: 400;">mongodb/mongodbatlas</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="/docs/registry/providers/images/community-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Community providers are published to the Terraform Registry by individual maintainers, groups of maintainers, or other members of the Terraform community.</td>
<td style="width: 31.7889%; height: 21px;"><br />Maintainer&rsquo;s individual or organization account, e.g. <code>DeviaVir/gsuite</code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="/docs/registry/providers/images/archived-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Archived Providers are Official or Verified Providers that are no longer maintained by HashiCorp or the community. This may occur if an API is deprecated or interest was low.</td>
<td style="width: 31.7889%; height: 21px;"><code>hashicorp</code> or third-party</td>
</tr>
</tbody>
</table>


## How to Develop Providers

Providers are written in Go, using the Terraform Plugin SDK. For more
information on developing providers, see:

- The [Extending Terraform](/docs/extend/index.html) documentation
- The [Call APIs with Terraform Providers](https://learn.hashicorp.com/collections/terraform/providers?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS)
  collection on HashiCorp Learn
