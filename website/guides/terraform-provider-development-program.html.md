---
layout: "extend"
page_title: "Terraform Provider Development Program"
sidebar_current: "guides-terraform-provider-development-program"
description: This guide is intended for vendors who're interested in having their platform supported by Terraform. The guide walks vendors through the steps involved in creating a provider and applying for it to be included with Terraform.
---

# Terraform Provider Development Program

The Terraform Provider Development Program facilitates vendors in creating and publishing Terraform providers that have been officially approved and verified by HashiCorp. Once verified, the provider published under your organization’s namespace will receive a distinct tier and badge that helps to distinguish it from community-sourced providers within the [Registry](https://registry.terraform.io).

The Verified badge helps users easily identify and discover integrations developed and maintained directly by an integration’s vendor, establishing a level of trust for our users. This program is intended to be largely self-serve, with links to information sources, clearly defined steps, and checkpoints detailed below.

![Verified Provider Card](/assets/images/docs/verified-card.png)

-> **Building your own provider?** If you're building your own provider and aren't interested in having HashiCorp officially verify and regularly monitor your provider, please refer to the [Call APIs with Terraform Providers](https://learn.hashicorp.com/collections/terraform/providers?utm_source=WEBSITEhttps://www.terraform.io/docs/extend/writing-custom-providers.htmlutm_medium=WEB_IOhttps://www.terraform.io/docs/extend/writing-custom-providers.htmlutm_offer=ARTICLE_PAGEhttps://www.terraform.io/docs/extend/writing-custom-providers.htmlutm_content=DOCS) collection on HashiCorp Learn and the [Extending Terraform](https://www.terraform.io/docs/extend/index.html) section of the documentation.


## What is a Terraform Provider?

Terraform is used to create, manage, and interact with infrastructure resources of any kind. Examples of resources include physical machines, VMs, network switches, containers, etc. Almost any infrastructure noun can be represented as a resource in Terraform.

A Terraform Provider represents an integration that is responsible for understanding API interactions with the underlying infrastructure, such as a public cloud service (AWS, GCP, Azure), a PaaS service (Heroku), a SaaS service (DNSimple, CloudFlare), or on-prem resources (vSphere). The Provider then exposes these as resources that Terraform users can interface with, from within Terraform a configuration. Terraform presently supports more than 70 providers, a number that has more than doubled in the past 12 months.

All providers integrate into and operate with Terraform exactly the same way. The table below is intended to help users understand who develops, and maintains a particular provider.

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
<p></p>


-> **Note:** This document focuses on the "Verified" Tier in the table above. Community contributors interested in contributing to existing providers or building new providers should refer to the [Publishing a Provider](https://www.terraform.io/docs/registry/providers/publishing.html) section of our documentation.


## Provider Development Process

The provider development process is divided into five steps below. By following these steps, providers can be developed alongside HashiCorp to ensure new providers are able to be published in Terraform as quickly as possible.

![Provider Development Process](/assets/images/docs/program-steps.png)

1. **Apply**: Initial contact between vendor and HashiCorp
2. **Prepare**: Follow documentation while developing the provider
3. **Verify**: Share public GPG key with HashiCorp
4. **Publish**: Release the provider on the Registry
5. **Support**: Ongoing maintenance and support of the provider by the vendor.

### 1. Apply

Please begin by completing our HashiCorp Technology Partner application: https://www.hashicorp.com/ecosystem/become-a-partner/#technology

Terraform has a large and active ecosystem of partners that may have already started working on the same provider. We'll do our best to connect similar parties to avoid duplicate efforts, and prepare for a successful and impactful launch of the integration. Once you have applied, a member of the HashiCorp Alliances team will be in touch, and will ask for your organization to sign our Technology Partner Agreement.


### 2. Prepare

Detailed instructions for preparing a provider for publishing are available in our Registry documentation. Please see [Preparing your Provider](https://www.terraform.io/docs/registry/providers/publishing.html#preparing-your-provider). In order to provide a consistent and quality experience for users, please make sure detailed documentation for your provider is included. You can find more information on how to build and structure [provider documentation here](https://www.terraform.io/docs/registry/providers/docs.html).

We’ve found the provider development process to be fairly straightforward and simple when you pay close attention and follow the resources below. If you have not developed a provider before and are looking for some help in developing one, you may choose to leverage one of the following development agencies which have developed Terraform providers in the past and are familiar with the requirements and process:

| Partner            | Email                        | Website              |
|--------------------|:-----------------------------|:---------------------|
| Crest Data Systems | malhar@crestdatasys.com      | www.crestdatasys.com |
| DigitalOnUs        | hashicorp@digitalonus.com    | www.digitalonus.com  |
| Akava              | bd@akava.io                  | www.akava.io         |
| OpenCredo          | hashicorp@opencredo.com      | www.opencredo.com    |

-> **Important:** All Terraform providers listed as Verified must contain one of the following open source licenses:

- CDDL 1.0, 2.0
- CPL 1.0
- Eclipse Public License (EPL) 1.0
- MPL 1.0, 1.1, 2.0
- PSL 2.0
- Ruby's Licensing
- AFL 2.1, 3.0
- Apache License 2.0
- Artistic License 1.0, 2.0
- Apache Software License (ASL) 1.1
- Boost Software License
- BSD, BSD 3-clause, "BSD-new"
- CC-BY
- Microsoft Public License (MS-PL)
- MIT


### 3. Verify

At this stage, it is expected that the provider is fully developed, all tests and documentation are in place, and your provider is ready for publishing. In this step, HashiCorp will verify the source and authenticity of the namespace being used to publish the provider by signing your GPG key with a trust signature.

-> **Important:** This step requires that you have signed and accepted our Technology Partner Agreement. If you have not received this, please see step #1 above.

Please send your public key to terraform-registry@hashicorp.com, indicating you are a partner seeking verification, and a HashiCorp employee will be in touch to help verify, and add your key.

To export your public key in ASCII-armor format, use the following command:

```
$ gpg --armor --export "{Key ID or email address}"
```

### 4. Publish

Once the verification step is complete please follow the steps on [Publishing a Provider](https://www.terraform.io/docs/registry/providers/publishing.html).  This step does not require additional involvement from HashiCorp as publishing is a fully self-service process in the [Terraform Registry](https://registry.terraform.io).

Once completed, your provider should be visible in the Terraform Registry and usable in Terraform. Please confirm that everything looks good, and that documentation is rendering properly.

### 5. Maintain & Support

Getting a new provider built and published to the Terraform Registry is just the first step towards enabling your users with a quality Terraform integration. Once a `verified` provider has been published, on-going effort is required to maintain the provider. It is expected that all verified provider publishers will continue to maintain the provider and address any issues your users report in a timely manner. HashiCorp reserves the right to remove verified status from any provider this is no longer maintained.

The expectation is to resolve all critical issues within 48 hours and all other issues within 5 business days. HashiCorp Terraform has an extremely wide community of users and contributors and we encourage everyone to report issues however small, as well as help resolve them when possible.

Vendors who choose to not support their provider and prefer to make it a community supported provider will no longer be listed as Verified.

## Contact Us

For any questions or feedback please contact us at <terraform-provider-dev@hashicorp.com>.
