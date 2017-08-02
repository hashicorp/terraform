---
layout: "guides"
page_title: "Terraform Provider Development Program"
sidebar_current: "guides-terraform-provider-development-program"
description: |-
  This guide provides steps to create a provider and apply for inclusing with
  Terraform, in order for Vendors to have their platform supported by Terraform.
---
#Terraform Provider Development Program

## Introduction
Terraform is used to create, manage, and manipulate infrastructure resources. Examples of resources include physical machines, VMs, network switches, containers, etc. Almost any infrastructure noun can be represented as a resource in Terraform.
 
Terraform can broadly be divided into two parts – the Terraform core, which consists of the core functionality, and a provider layer, which provides a translation layer between Terraform core and the underlying infrastructure. A provider is responsible for understanding API interactions with the underlying infrastructure like a cloud (AWS, GCP, Azure), a PaaS service (Heroku), a SaaS (service DNSimple, CloudFlare), or on-prem resources (vSphere). It then exposes these as resources users can code to. Terraform presently supports more than 70 providers, a number that has more than doubled in the past 12 months.

~> **NOTE:** This document is intended for vendors and users who would like to build a Terraform provider to have their infrastructure supported via terraform.  The program is intended to be largely self-serve, with links to information sources, clearly defined steps, and checkpoints. This being said, we welcome you to contact us at <terraform-provider-dev@hashicorp.com> with any questions, or feedback.


## Provider Development Process
The Terraform provider development process can broadly be divided into the six steps described below.

![Process](docs/process.png)

1. Engage:    Initial contact between vendor and HashiCorp
2. Enable:    Information and articles to aid with the provider development
3. Dev/Test:  Provider development and test process
4. Review:    HashiCorp code review and acceptance tests (iterative process)
5. Release:   Provider availability and listing on [terraform.io](https://www.terraform.io)
6. Support:   Ongoing maintenance and support of the provider by the vendor.

Each of these steps are described in detail below. Vendors are encouraged to follow the tasks associated with each step to the fullest as it helps streamline the process and minimize rework.

### 1. Engage
Each new provider development cycle begins with the vendor providing some basic information about the infrastructure the provider is being built for, the name of the provider, relevant details about the project, via a simple [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2) (https://goo.gl/forms/iqfz6H9UK91X9LQp2).  This information is captured upfront and used for consistently tracking the provider through the various steps.
 
All providers integrate into and operate with Terraform exactly the same way.  The table below is intended to help users understand who develops, maintains and tests a particular provider. All new providers should align to one of these two tiers.

![Engage-table](docs/engage-table.png)

### 2. Enable
In order to get started with the Terraform provider development process we recommend reviewing and following the articles included in the Provider Development Kit.
 
Provider Development Kit:

* Writing custom providers [guide](https://www.terraform.io/guides/writing-custom-terraform-providers.html)
* How-to build a provider [video](https://www.youtube.com/watch?v=2BvpqmFpchI)
* Sample provider developed by [partner](http://container-solutions.com/write-terraform-provider-part-1/)
* Example providers for reference: [AWS](https://github.com/terraform-providers/terraform-provider-aws), [OPC](https://github.com/terraform-providers/terraform-provider-opc)
* Contributing to Terraform [guidelines](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md)
* Gitter HashiCorp-Terraform [room](https://gitter.im/hashicorp-terraform/Lobby).
 
We’ve found the provider development to be fairly straightforward and simple when vendors pay close attention and follow to the above articles. Adopting the same structure and coding patterns helps expedite the review and release cycles.

### 3. Development & Test
The Terraform provider is written in the [Go](https://golang.org/) programming language. The best approach to architect a new provider project is to use the [AWS provider](https://github.com/terraform-providers/terraform-provider-aws) as a reference.  Given the wide surface area of this provider, almost all resource types and preferred code constructs are covered in it.
 
It is recommended for vendors to first develop support for one or two resources and go through an initial review cycle before developing the code for the remaining resources.  This helps catch any issues early on in the process and avoids errors from getting multiplied. In addition, it is advised to follow existing conventions you see in the codebase, and ensure your code is formatted with go fmt.  This is needed as our TravisCI continuous Integration (CI) build will fail if go fmt has not been run on the code.
 
The provider code should include an acceptance test suite with tests for each individual resource that holistically tests its behavior. The Writing Acceptance Tests section in the [Contributing to Terraform](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md) document explains how to approach these. It is recommended to randomize the names of the tests as opposed to using unique static names, as that permits us to parallelize the test execution.


Another common problem is that those tests can only run one at a time (because of unique static names of resources).
Randomized names is what we should encourage people to do.


Each provider has a section in the Terraform documentation. You'll want to add new index file and individual pages for each resource supported by the provider.
 
While developing the provider code yourself is certainly possible, you can also choose to leverage one of the following development agencies who’ve developed Terraform providers in the past and are familiar with the requirements and process.

| Partner            | Website                      | Email                |
|:-------------------|:-----------------------------|:---------------------|
| Crest Data Systems | malhar@crestdatasys.com      | www.crestdatasys.com |
| DigitalOnUs        | hashicorp@digitalonus.com    | www.digitalonus.com  |
| MustWin            | bd@mustwin.com               | www.mustwin.com      |
| OpenCredo          | guy.richardson@opencredo.com | www.opencredo.com    |

### 4. Review
Once the provider with one or two sample resources has been developed, an email should be sent to <terraform-provider-dev@hashicorp.com> along with a pointer to the public GitHub repo containing the code. HashiCorp will then review the resource code, acceptance tests, and the documentation for the sample resource(s) will be reviewed. When all the feedback has been addressed, support for the remaining resources can continue to be developed, along with the corresponding acceptance tests and documentation. The vendor is encouraged to send HashiCorp a rough list of resource names that are planned to be worked on along with the mapping to the underlying APIs, if possible.  This information can be provided via the [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2). It is preferred that the additional resources be developed and submitted as individual PRs in GitHub as that simplifies the review process.
 
Once the provider has been completed another email should be sent to <terraform-provider-dev@hashicorp.com> along with a pointer to the public GitHub repo containing the code requesting the final code review. HashiCorp will review the code and provide feedback about any changes that may be required.  This is often an iterative process and can take some time to get done.
 
The vendor is also required to provide access credentials for the infrastructure (cloud or other) that is managed by the provider. Please encrypt the credentials using our public GPG key published at keybase.io/terraform (you can use the form at https://keybase.io/encrypt#terraform) and paste the encrypted message into the [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2). Please do NOT enter plain-text credentials. These credentials are used during the review phase, as well as to test the provider as part of the regular testing HashiCorp conducts.
 
>
NOTE: It is strongly recommended to develop support for just one or two resources first and go through the review cycle before developing support for all the remaining resources. This approach helps catch any code construct issues early, and avoids the problem from multiplying across other resources.  In addition, one of the common gaps is often the lack of a complete set of acceptance tests, which results in wasted time. It is recommended that you make an extra pass through the provider code and ensure that each resource has an acceptance test associated with it.

### 5. Release
At this stage, it is expected that the provider is fully developed, all tests and documentation are in place, and the acceptance tests are all passing.


HashiCorp will create a new GitHub repo under the terraform-providers GitHub organization for the new provider (example: terraform-providers/terraform-provider-_name_) and grant the owner of the original provider code write access to the new repo. A GitHub Pull Request should be created against this new repo with the provider code that had been reviewed in step-4 above. Once this is done HashiCorp will review and merge the PR, and get the new provider listed on [terraform.io](https://www.terraform.io). This is also when the provider acceptance tests are added to the HashiCorp test harness (TeamCity) and tested at regular intervals.


Vendors whose providers are listed on terraform.io are permitted to use the HashiCorp Tested logo for their provider.

<img alt="hashicorp-tested-icon" src="/assets/images/docs/hashicorp-tested-icon.png" style="width: 101px;" />

### 6. Support
Many vendors view the Release step above to be the end of the journey, while at HashiCorp we view it to be the start. Getting the provider built is just the first step in enabling users to use it against the infrastructure. Once this is done on-going effort is required to maintain the provider and address any issues in a timely manner. The expectation is to resolve all Critical issues within 48 hours and all other issues within 5 business days. HashiCorp Terraform has as extremely wide community of users and contributors and we encourage everyone to report issues however small, as well as help resolve them when possible.
 
Vendors who choose to not support their provider and prefer to make it a community supported provider will not be listed on terraform.io.

## Next Steps
Below is an ordered checklist of steps that should be followed during the provider development process.

[ ] Fill out provider development program engagement [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2) (https://goo.gl/forms/iqfz6H9UK91X9LQp2)

[ ] Refer to the example providers and model the new provider based on that

[ ] Create the new provider with one or two sample resources along with acceptance tests and documentation

[ ] Send email to <terraform-provider-dev@hashicorp.com> to schedule an initial review

[ ] Address review feedback and develop support for the other resources

[ ] Send email to <terraform-provider-dev@hashicorp.com> along with a pointer to the public GitHub repo containing the final code

[ ] Provide HashiCorp with credentials for underlying infrastructure managed by the new provider via the [webform](https://goo.gl/forms/iqfz6H9UK91X9LQp2)

[ ] Address all review feedback, ensure that each resource has a corresponding  acceptance test, and the documentation is complete

[ ] Create a PR for the provider against the HashiCorp provided empty repo.

[ ] Plan to continue supporting the provider with additional functionality as well as addressing any open issues.


In this document we’ve covered the process for getting a Terraform provider created. For any questions or feedback please contact us at <terraform-provider-dev@hashicorp.com>.
