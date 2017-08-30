---
layout: "guides"
page_title: "Terraform Provider Development Program"
sidebar_current: "guides-terraform-provider-development-program"
description: This guide is intended for vendors who're interested in having their platform supported by Teraform. The guide walks vendors through the steps involved in creating a provider and applying for it to be included with Terraform.
---

# Terraform Provider Development Program

The Terraform Provider Development Program allows vendors to build
Terraform providers that are officially approved and tested by HashiCorp and
listed on the official Terraform website. The program is intended to be largely
self-serve, with links to information sources, clearly defined steps, and
checkpoints.

-> **Building your own provider?** If you're building your own provider and
aren't interested in having HashiCorp officially approve and regularly test
the provider, refer to the
[Writing Custom Providers guide](/guides/writing-custom-terraform-providers.html).

## What is a Terraform Provider?

Terraform is used to create, manage, and manipulate infrastructure resources.
Examples of resources include physical machines, VMs, network switches, containers, etc.
Almost any infrastructure noun can be represented as a resource in Terraform.

A provider is responsible for understanding API interactions with the underlying
infrastructure like a cloud (AWS, GCP, Azure), a PaaS service (Heroku), a SaaS
(service DNSimple, CloudFlare), or on-prem resources (vSphere). It then exposes
these as resources users can code to. Terraform presently supports more than
70 providers, a number that has more than doubled in the past 12 months.

All providers integrate into and operate with Terraform exactly the same way.
The table below is intended to help users understand who develops, maintains
and tests a particular provider.

![Provider Engagement Table](/assets/images/docs/engage-table.png)

-> **Note:** This document is primarily intended for the "HashiCorp/Vendors" row in
the table above. Community contributors who’re interested in contributing to
existing providers or building new providers should refer to the
[Writing Custom Providers guide](/guides/writing-custom-terraform-providers.html).

## Provider Development Process

The provider development process is divided into six steps below. By following
these steps, providers can be developed alongside HashiCorp to ensure new
providers are able to be published in Terraform as quickly as possible.

![Provider Development Process](/assets/images/docs/process.png)

1. **Engage**:    Initial contact between vendor and HashiCorp
2. **Enable**:    Information and articles to aid with the provider development
3. **Dev/Test**:  Provider development and test process
4. **Review**:    HashiCorp code review and acceptance tests (iterative process)
5. **Release**:   Provider availability and listing on [terraform.io](https://www.terraform.io)
6. **Support**:   Ongoing maintenance and support of the provider by the vendor.

### 1. Engage

Please begin by providing some basic information about the provider that
is being built via a simple [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2).

This information is captured upfront and used by HashiCorp to track the
provider through various stages. The information is also used to notify the
provider developer of any overlapping work, perhaps coming from the community.

Terraform has a large and active community and ecosystem of partners that
may have already started working on the same provider. We'll do our best to
connect similar parties to avoid duplicate work.

### 2. Enable

We’ve found the provider development to be fairly straightforward and simple
when vendors pay close attention and follow to the resources below. Adopting
the same structure and coding patterns helps expedite the review and release cycles.

* Writing custom providers [guide](https://www.terraform.io/guides/writing-custom-terraform-providers.html)
* How-to build a provider [video](https://www.youtube.com/watch?v=2BvpqmFpchI)
* Sample provider developed by [partner](http://container-solutions.com/write-terraform-provider-part-1/)
* Example providers for reference: [AWS](https://github.com/terraform-providers/terraform-provider-aws), [OPC](https://github.com/terraform-providers/terraform-provider-opc)
* Contributing to Terraform [guidelines](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md)
* Gitter HashiCorp-Terraform [room](https://gitter.im/hashicorp-terraform/Lobby).

### 3. Development & Test

Terraform providers are written in the [Go](https://golang.org/) programming
language. The
[Writing Custom Providers guide](/guides/writing-custom-terraform-providers.html)
is a good resource for developers to begin writing a new provider.

The best approach to building a new provider project is to use the
[AWS provider](https://github.com/terraform-providers/terraform-provider-aws)
as a reference.  Given the wide surface area of this provider, almost all
resource types and preferred code constructs are covered in it.

It is recommended for vendors to first develop support for one or two resources
and go through an initial review cycle before developing the code for the
remaining resources.  This helps catch any issues early on in the process and
avoids errors from getting multiplied. In addition, it is advised to follow
existing conventions you see in the codebase, and ensure your code is formatted
with `go fmt`.

The provider code should include an acceptance test suite with tests for each
individual resource that holistically tests its behavior.
The Writing Acceptance Tests section in the
[Contributing to Terraform](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md)
document explains how to approach these. It is recommended to randomize the
names of the tests as opposed to using unique static names, as that permits us
to parallelize the test execution.

Each provider has a section in the Terraform documentation. You'll want to add
new index file and individual pages for each resource supported by the provider.

While developing the provider code yourself is certainly possible, you can also
choose to leverage one of the following development agencies who’ve developed
Terraform providers in the past and are familiar with the requirements and process.

| Partner            | Email                        | Website              |
|--------------------|:-----------------------------|:---------------------|
| Crest Data Systems | malhar@crestdatasys.com      | www.crestdatasys.com |
| DigitalOnUs        | hashicorp@digitalonus.com    | www.digitalonus.com  |
| MustWin            | bd@mustwin.com               | www.mustwin.com      |
| OpenCredo          | hashicorp@opencredo.com      | www.opencredo.com    |

### 4. Review

During the review process, HashiCorp will provide feedback on the newly
developed provider. **Please engage in the review process once one or two
sample resources have been developed.** Begin the process by emailing
<terraform-provider-dev@hashicorp.com> with a URL to the public GitHub repo
containing the code.

HashiCorp will then review the resource code, acceptance tests, and the
documentation. When all the feedback has been addressed, support for the
remaining resources can continue to be developed, along with the corresponding
acceptance tests and documentation.

The vendor is encouraged to send HashiCorp
a rough list of resource names that are planned to be worked on along with the
mapping to the underlying APIs, if possible.  This information can be provided
via the [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2). It is preferred that
the additional resources be developed and submitted as individual PRs in GitHub
as that simplifies the review process.

Once the provider has been completed another email should be sent to
<terraform-provider-dev@hashicorp.com> along with a URL to the public GitHub repo
containing the code requesting the final code review. HashiCorp will review the
code and provide feedback about any changes that may be required.  This is often
an iterative process and can take some time to get done.

The vendor is also required to provide access credentials for the infrastructure
(cloud or other) that is managed by the provider. Please encrypt the credentials
using our public GPG key published at keybase.io/terraform (you can use the form
at https://keybase.io/encrypt#terraform) and paste the encrypted message into
the [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2). Please do NOT enter
plain-text credentials. These credentials are used during the review phase,
as well as to test the provider as part of the regular testing HashiCorp conducts.

->
**NOTE:** It is strongly recommended to develop support for just one or two resources first and go through the review cycle before developing support for all the remaining resources. This approach helps catch any code construct issues early, and avoids the problem from multiplying across other resources.  In addition, one of the common gaps is often the lack of a complete set of acceptance tests, which results in wasted time. It is recommended that you make an extra pass through the provider code and ensure that each resource has an acceptance test associated with it.

### 5. Release

At this stage, it is expected that the provider is fully developed, all tests
and documentation are in place,the acceptance tests are all passing, and that
HashiCorp has reviewed the provider.

HashiCorp will create a new GitHub repo under the terraform-providers GitHub
organization for the new provider (example: `terraform-providers/terraform-provider-NAME`)
and grant the owner of the original provider code write access to the new repo.
A GitHub Pull Request should be created against this new repo with the provider
code that had been reviewed in step-4 above. Once this is done HashiCorp will
review and merge the PR, and get the new provider listed on
[terraform.io](https://www.terraform.io). This is also when the provider
acceptance tests are added to the HashiCorp test harness (TeamCity) and tested
at regular intervals.

Vendors whose providers are listed on terraform.io are permitted to use the
[HashiCorp Tested logo](/assets/images/docs/hashicorp-tested-icon.png) for their provider.

### 6. Support

Many vendors view the release step to be the end of the journey, while at
HashiCorp we view it to be the start. Getting the provider built is just the
first step in enabling users to use it against the infrastructure. Once this is
done on-going effort is required to maintain the provider and address any
issues in a timely manner.

The expectation is to resolve all critical issues within 48 hours and all other
issues within 5 business days. HashiCorp Terraform has as extremely wide
community of users and contributors and we encourage everyone to report issues
however small, as well as help resolve them when possible.

Vendors who choose to not support their provider and prefer to make it a
community supported provider will not be listed on terraform.io.

## Checklist

Below is an ordered checklist of steps that should be followed during the
provider development process. This just reiterates the steps already documented
in the section above.

* Fill out provider development program engagement [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2)

* Refer to the example providers and model the new provider based on that

* Create the new provider with one or two sample resources along with acceptance tests and documentation

* Send email to <terraform-provider-dev@hashicorp.com> to schedule an initial review

* Address review feedback and develop support for the other resources

* Send email to <terraform-provider-dev@hashicorp.com> along with a pointer to the public GitHub repo containing the final code

* Provide HashiCorp with credentials for underlying infrastructure managed by the new provider via the [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2)

* Address all review feedback, ensure that each resource has a corresponding  acceptance test, and the documentation is complete

* Create a PR for the provider against the HashiCorp provided empty repo.

* Plan to continue supporting the provider with additional functionality as well as addressing any open issues.

## Contact Us

For any questions or feedback please contact us at <terraform-provider-dev@hashicorp.com>.
