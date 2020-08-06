---
layout: "registry"
page_title: "Terraform Registry - Providers Overview"
description: |-
  Overview of providers in the Terraform Registry
---

# Overview

 The Registry offers both a place for users to find Providers, and also acts as the Public origin source for Terraform, meaning that all published providers are directly available from within the Terraform CLI. Providers are how Terraform integrates with any upstream API, whether another HashiCorp technology, or the many hundreds of third-party services that Terraform integrates with today. Creating a Provider is designed to be easy and intuitive, and the Registry is here to help you share it with the rest of the community.

## Provider Tiers & Namespaces

Terraform Providers are published and maintained by a variety of sources, including HashiCorp, HashiCorp Technology Partners, and the Terraform Community. Tiers and Badges are used in the Registry to denote the source of the Provider. Additionally, namespaces are used to help users identify the organization or publisher responsible for the integration, as shown in the table below.

<table border="0" style="border-collapse: collapse; width: 100%;">
<tbody>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><strong>Tier</strong></td>
<td style="width: 55.7271%; height: 21px;"><strong>Description</strong></td>
<td style="width: 31.7889%; height: 21px;"><strong>Namespace</strong></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/official-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;"><i><span style="font-weight: 400;">Official Providers are owned and maintained by HashiCorp </span></i></td>
<td style="width: 31.7889%; height: 21px;"><code><span style="font-weight: 400;">HashiCorp</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/verified-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;"><i><span style="font-weight: 400;">Verified Providers are owned and maintained by third-party technology partners. Providers in this tier indicate HashiCorp has verified the authenticity of the Provider&rsquo;s publisher, and that the partner is a member of the </span></i><a href="https://www.hashicorp.com/ecosystem/become-a-partner/"><i><span style="font-weight: 400;">HashiCorp Technology Partner Program</span></i></a><i><span style="font-weight: 400;">.</span></i></td>
<td style="width: 31.7889%; height: 21px;"><span style="font-weight: 400;">Third Party Organization, e.g. </span><code><span style="font-weight: 400;">mongodb/mongodbatlas</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/community-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Community Providers are published to the Terraform Registry by individual maintainers, groups of maintainers, or other members of the Terraform Community.</td>
<td style="width: 31.7889%; height: 21px;"><br />Maintainer&rsquo;s individual or organization, e.g. <code>DeviaVir/gsuite</code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="./images/archived-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Archived Providers are Official or Verified Providers that are no longer maintained by HashiCorp or the community. This may occur if an API is deprecated or interest was low.</td>
<td style="width: 31.7889%; height: 21px;"><code>HashiCorp</code> or Third-Party</td>
</tr>
</tbody>
</table>
<p></p>

## Verified Provider Development Program

For any organization interested in joining our Provider Development Program, indicated with a `Verified` badge on published providers or modules, please take a look at our [Program Details](https://www.terraform.io/guides/terraform-provider-development-program.html) for further information.
