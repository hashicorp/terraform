---
layout: "registry"
page_title: "Terraform Registry - Providers Overview"
description: |-
  Overview of providers in the Terraform Registry
---

# Overview

Providers are how Terraform integrates with any upstream API.

The Terraform Registry is the main source for publicly available Terraform providers. It offers a browsable and searchable interface for finding providers, and makes it possible for Terraform CLI to automatically install any of the providers it hosts.

If you want Terraform to support a new infrastructure service, you can create your own provider using Terraform's Go SDK. Once you've developed a provider, you can use the Registry to share it with the rest of the community.

## Using Providers From the Registry

The Registry is directly integrated with Terraform. To use any provider from the Registry, all you need to do is require it within your Terraform configuration; Terraform can then automatically install that provider when initializing a working directory, and your configuration can take advantage of any resources implemented by that provider.

For more information, see:

- [Configuration Language: Provider Requirements](/docs/configuration/provider-requirements.html)

## Provider Tiers & Namespaces

Terraform providers are published and maintained by a variety of sources, including HashiCorp, HashiCorp Technology Partners, and the Terraform community. The Registry uses tiers and badges to denote the source of a provider. Additionally, namespaces are used to help users identify the organization or publisher responsible for the integration, as shown in the table below.

<table border="0" style="border-collapse: collapse; width: 100%;">
<tbody>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><strong>Tier</strong></td>
<td style="width: 55.7271%; height: 21px;"><strong>Description</strong></td>
<td style="width: 31.7889%; height: 21px;"><strong>Namespace</strong></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/official-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;"><i><span style="font-weight: 400;">Official providers are owned and maintained by HashiCorp </span></i></td>
<td style="width: 31.7889%; height: 21px;"><code><span style="font-weight: 400;">hashicorp</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/verified-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;"><i><span style="font-weight: 400;">Verified providers are owned and maintained by third-party technology partners. Providers in this tier indicate HashiCorp has verified the authenticity of the Provider&rsquo;s publisher, and that the partner is a member of the </span></i><a href="https://www.hashicorp.com/ecosystem/become-a-partner/"><i><span style="font-weight: 400;">HashiCorp Technology Partner Program</span></i></a><i><span style="font-weight: 400;">.</span></i></td>
<td style="width: 31.7889%; height: 21px;"><span style="font-weight: 400;">Third-party organization, e.g. </span><code><span style="font-weight: 400;">mongodb/mongodbatlas</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/community-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Community providers are published to the Terraform Registry by individual maintainers, groups of maintainers, or other members of the Terraform community.</td>
<td style="width: 31.7889%; height: 21px;"><br />Maintainer&rsquo;s individual or organization account, e.g. <code>DeviaVir/gsuite</code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/archived-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Archived Providers are Official or Verified Providers that are no longer maintained by HashiCorp or the community. This may occur if an API is deprecated or interest was low.</td>
<td style="width: 31.7889%; height: 21px;"><code>hashicorp</code> or third-party</td>
</tr>
</tbody>
</table>
<p></p>

## Verified Provider Development Program

If your organization is interested in joining our Provider Development Program (which sets the standards for publishing providers and modules with a `Verified` badge), please take a look at our [Program Details](/guides/terraform-provider-development-program.html) for further information.
