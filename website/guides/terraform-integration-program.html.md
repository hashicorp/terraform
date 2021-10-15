---
layout: "extend"
page_title: "Terraform Integration Program"
sidebar_current: "guides-terraform-integration-program"
description: The Terraform Integration Program allows prospect partners to create and publish Terraform integrations that have been verified by HashiCorp.
---

# Terraform Integration Program

The Terraform Integration Program facilitates prospect partners in creating and publishing Terraform integrations that have been verified by HashiCorp. 

## Terraform Editions

Terraform is an infrastructure as code (IaC) tool that allows you to build, change, and version infrastructure safely and efficiently. This includes low-level components such as compute instances, storage, and networking, as well as high-level components such as DNS entries, SaaS features, etc. Terraform can manage both existing service providers and custom in-house solutions.

HashiCorp offers three editions of Terraform: Open Source, Terraform Cloud, and Terraform Enterprise.

- [Terraform Open Source](https://www.terraform.io/) provides a consistent CLI workflow to manage hundreds of cloud services. Terraform codifies cloud APIs into declarative configuration files.
- [Terraform Cloud (TFC)](https://www.terraform.io/cloud) is a free to use, self-service SaaS platform that extends the capabilities of the open source Terraform CLI. It adds automation and collaboration features, and performs Terraform functionality remotely, making it ideal for collaborative and production environments. Terraform Cloud is available as a hosted service at https://app.terraform.io. Small teams can sign up for free to connect Terraform to version control, share variables, run Terraform in a stable remote environment, and securely store remote state. Paid tiers allow you to add more than five users, create teams with different levels of permissions, enforce policies before creating infrastructure, and collaborate more effectively.
- [Terraform Enterprise (TFE)](https://www.terraform.io/docs/enterprise/index.html) is our self-hosted distribution of Terraform Cloud with advanced security and compliance features. It offers enterprises a private instance that includes the advanced features available in Terraform Cloud.

## Types of Terraform Integrations

The Terraform ecosystem is designed to enable users to apply Terraform across different use cases and environments. The Terraform Integration Program current supports both workflow and integration partners (details below). Some partners can be both, depending on their use cases.  
- **Workflow Partners** build integrations for Terraform Cloud and/or Terraform Enterprise.  Ideally, these partners are seeking to enable customers to use their existing platform within a Terraform Run. 
- **Infrastructure Partners** empower customers to leverage Terraform to manage resources exposed by their platform APIs. These are accessible to users of all Terraform editions.

Our Workflow Partners typically have the following use cases:

- **Code Scanning:** These partners provide tooling to review infrastructure as code configurations to prevent errors or security issues.
- **Cost Estimation:** These partners drive cost estimation of new deployment based on historical deployments.
- **Monitoring:** These partners provide performance visibility.
- **Zero Trust Security:** These partners help users create configurations to verify connections prior to providing access to an organization’s systems.
- **Audit:** These partners focus on maintaining code formatting, preventing security threats, and performing additional code analysis.
- **ITSM (Information Technology Service Management):** These partners focus on implementation, deployment, and delivery of IT workflows.
- **SSO (Single Sign On):** These partners focus on authentication for end users to securely sign on.
- **CI/CD:** These partners focus on continuous integration and continuous delivery/deployment.
- **VCS:** These partners focus on tracking and managing software code changes.

Most workflow partners integrate with the Terraform workflow itself. Run tasks allow Terraform Cloud to execute tasks in external systems at specific points in the Terraform Cloud run lifecycle. This offers much more extensibility to Terraform Cloud customers, enabling them to integrate your services into the Terraform Cloud workflow. The beta release of this feature allows users to add and execute these tasks during the new pre-apply stage which exists in between the plan and apply stages. Eventually, we will open the entire workflow to Terraform Cloud users, including the pre-plan and post apply stages. Reference the [Terraform Cloud Integrations documentation](https://www.terraform.io/guides/terraform-integration-program.html#terraform-cloud-integrations) for more details.

![Integration program diagram](/assets/images/docs/terraform-integration-program-diagram.png)

Our Infrastructure Partners typically have the following use cases:

- **Public Cloud:** These are large-scale, global cloud providers that offer a range of services including IaaS, SaaS, and PaaS.
- **Container Orchestration:** These partners help with container provisioning and deployment.
- **IaaS (Infrastructure-as-a-Service):** These are infrastructure and IaaS providers that offer solutions such as storage, networking, and virtualization.
- **Security & Authentication:** These are partners with authentication and security monitoring platforms.
- **Asset Management:** These partners offer asset management of key organization and IT resources, including software licenses, hardware assets, and cloud resources.
- **CI/CD:** These partners focus on continuous integration and continuous delivery/deployment.
- **Logging & Monitoring:** These partners offer the capability to configure and manage services such as loggers, metric tools, and monitoring services.
- **Utility:** These partners offer helper functionality, such as random value generation, file creation, http interactions, and time-based resources.
- **Cloud Automation:** These partners offer specialized cloud infrastructure automation management capabilities such as configuration management.
- **Data Management:** These partners focus on data center storage, backup, and recovery solutions.
- **Networking:** These partners integrate with network-specific hardware and virtualized products such as routing, switching, firewalls, and SD-WAN solutions.
- **VCS (Version Control Systems):** These partners focus on VCS (Version Control System) projects, teams, and repositories from within Terraform.
- **Comms & Messaging:** These partners integrate with communication, email, and messaging platforms.
- **Database:** These partners offer capabilities to provision and configure your database resources.
- **PaaS (Platform-as-a-Service):** These are platform and PaaS providers that offer a range of hardware, software, and application development tools. This category includes smaller-scale providers and those with more specialized offerings.
- **Web Services:** These partners  focus on web hosting, web performance, CDN and DNS services.

Infrastructure partners integrate by building and publishing a plugin called a Terraform [provider](https://www.terraform.io/docs/language/providers/index.html). Providers are executable binaries written in Go that communicate with Terraform Core over an RPC interface. The provider acts as a translation layer for transactions with external APIs, such as a public cloud service (AWS, GCP, Azure), a PaaS service (Heroku), a SaaS service (DNSimple, CloudFlare), or on-prem resources (vSphere). Providers work across Terraform OSS, Terraform Cloud and Terraform Enterprise. Refer to the [Terraform Provider Integrations documentation](https://www.terraform.io/guides/terraform-integration-program.html#terraform-provider-integrations) for more detail.



## Terraform Provider Integrations

You can follow the five steps. below to develop your provider alongside HashiCorp. This ensures that you can publish new versions with Terraform quickly and efficiently. 

![Provider Development Process](/assets/images/docs/provider-program-steps.png)

1. **Prepare**: Develop integration using included resources
2. **Publish**: Publish provider to the Registry or plugin documentation
3. **Apply**: Apply to Technology Partnership Program
4. **Verify**: Verify integration with HashiCorp Alliances team
5. **Support**: Ongoing maintenance and support of the integration by the vendor

Each of these steps are described in detail below. Partners are encouraged to follow the tasks associated with each step to the fullest as it helps streamline the process and minimize rework.

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
<td style="width: 31.7889%; height: 21px;"><span style="font-weight: 400;">Third-party organization, e.g. </span><code><span style="font-weight: 400;">cisco/aci</span></code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="/docs/registry/providers/images/community-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Community providers are published to the Terraform Registry by individual maintainers, groups of maintainers, or other members of the Terraform community.</td>
<td style="width: 31.7889%; height: 21px;"><br />Maintainer&rsquo;s individual or organization account, e.g. <code>cyrilgdn/postgresql</code></td>
</tr>
<tr style="height: 21px;">
<td style="width: 12.4839%; height: 21px;"><img src="/docs/registry/providers/images/archived-tier.png" alt="" /></td>
<td style="width: 55.7271%; height: 21px;">Archived Providers are Official or Verified Providers that are no longer maintained by HashiCorp or the community. This may occur if an API is deprecated or interest was low.</td>
<td style="width: 31.7889%; height: 21px;"><code>hashicorp</code> or third-party</td>
</tr>
</tbody>
</table>


### 1. Prepare
To get started with the Terraform provider development, we recommend reviewing and following the articles listed below.
#### Provider Development Kit

a) Writing custom providers [guide](https://www.terraform.io/guides/writing-custom-terraform-providers.html)

b) Creating a Terraform Provider for Just About Anything: [video](https://www.youtube.com/watch?v=noxwUVet5RE)

c) Sample provider developed by [partner](http://container-solutions.com/write-terraform-provider-part-1/)

d) Example provider for reference: [AWS](https://github.com/terraform-providers/terraform-provider-aws), [OPC](https://github.com/terraform-providers/terraform-provider-opc)

e) Contributing to Terraform [guidelines](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md)

f) HashiCorp developer [forum](https://discuss.hashicorp.com/c/terraform-providers/tf-plugin-sdk/43)

We’ve found the provider development process to be fairly straightforward and simple when you pay close attention and follow the resources below. If you have not developed a provider before and are looking for some help in developing one, you may choose to leverage one of the following development agencies which have developed Terraform providers in the past and are familiar with the requirements and process:

| Partner            | Email                        | Website              |
|--------------------|:-----------------------------|:---------------------|
| Crest Data Systems | malhar@crestdatasys.com      | www.crestdatasys.com |
| DigitalOnUs        | hashicorp@digitalonus.com    | www.digitalonus.com  |
| Akava              | bd@akava.io                  | www.akava.io         |
| OpenCredo          | hashicorp@opencredo.com      | www.opencredo.com    |

#### Provider License

All Terraform providers listed as Verified must contain one of the following open source licenses:

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


-> **Note:** If you have questions or suggestions about the Terraform SDK and the development of the Terraform provider, please submit your request to the HashiCorp Terraform plugin SDK forum


### 2. Publish

After your provider development is complete and ready to release, vendors will publish the integration to the [Terraform Registry](https://registry.terraform.io/) for all Terraform users to discover by following the [publishing documentation](https://www.terraform.io/docs/registry/providers/publishing.html), or reviewing the [provider publishing learn guide](https://learn.hashicorp.com/tutorials/terraform/provider-release-publish?in=terraform/providers).

Once completed, your provider should be visible in the Terraform Registry and usable in Terraform. Please confirm that everything looks good, and that documentation is rendering properly.


-> **Note:** If your company has multiple products with separate providers, we recommend publishing them under the same Github organization to help with the discoverability.

### 3. Apply

Vendors should now connect with HashiCorp Alliances to onboard your integration to the HashiCorp technology ecosystem or apply to become a technology partner through [this form](https://www.hashicorp.com/ecosystem/become-a-partner/#technology).

### 4. Verify

Once the provider is published, vendors should work with their HashiCorp Alliances representative to verify the plugin within the Registry and listed as an HashiCorp technology partner integration on HashiCorp website.

### 5. Support

Getting a new provider built and published to the Terraform Registry is just the first step towards enabling your users with a quality Terraform integration. Once a verified provider has been published, on-going effort is required to maintain the provider. 

HashiCorp Terraform has an extremely wide community of users and contributors and we encourage everyone to report issues however small, as well as help resolve them when possible. We expect that all verified provider publishers will continue to maintain the provider and address any issues users report in a timely manner. This includes resolving all critical issues within 48 hours and all other issues within 5 business days. HashiCorp reserves the right to remove verified status from any integration that is no longer being maintained.

Vendors who choose not to support their provider and prefer to make it a community-supported provider will no longer be listed as Verified.

## Terraform Cloud Integrations

As indicated earlier, run tasks allow Terraform Cloud to execute tasks in external systems at specific points in the Terraform Cloud run lifecycle. The beta release of this feature allows users to add and execute these tasks during the new pre-apply stage* which exists in between the plan and apply stages. Tasks are executed by sending an API payload to the external system. This payload contains a collection of run-related information and a callback URL which the external system can use to send updates back to Terraform Cloud.

The external system can then use this run information and respond back to Terraform Cloud with a passed or failed status. Terraform Cloud uses this status response to determine if a run should proceed, based on the task's enforcement settings within a workspace.

Partners who successfully complete the Terraform Cloud Integration Checklist will obtain a Terraform Cloud badge. This signifies HashiCorp has verified the integration and the partner is a member of the HashiCorp Technology Partner Program. 

![TFC Badge](/assets/images/docs/tfc-badge.png)

The above badge will help drive visibility for the partner as well as provide better differentiation for joint customers. This badge will be available for partners to use at their digital properties (as per guidelines in the technology partner guide that partners receive when they join HashiCorp’s technology partner program).  

- Note: Currently, pre-apply is the only integration phase available at this time. As of September 2021, run tasks are available only as a beta feature, are subject to change, and not all customers will see this functionality in their Terraform Cloud organization since this is currently enabled by default for our business tier customers of Terraform Cloud. If you have a customer that is interested in run tasks and are not a current Terraform Cloud for Business customer, customers can [sign up here](https://docs.google.com/forms/d/e/1FAIpQLSf3JJIkU05bKWov2wXa9c-QV524WNaHuGIk7xjHnwl5ceGw2A/viewform). 

The Terraform Cloud Integration portion of this program is divided into five steps below.

![RunTask Program Process](/assets/images/docs/runtask-program-steps.png)

1. **Engage**: Interested partner should sign up for the Technology Partner
Program
2. **Develop & Test**: Understand and build using the API integration for Run Tasks 
3. **Review**: Review integration with HashiCorp Alliances team
4. **Release**: Provide documentation for your Integration
5. **Support**: Vendor provides ongoing maintanance and support

### 1. Engage

For partners who are new to working with Hashicorp, we recommend [signing up for our Technology Partner Program](https://www.hashicorp.com/go/tech-partner). To understand more about the program, check out our “[Become a Partner](https://www.hashicorp.com/partners/become-a-partner)” page.

### 2. Develop & Test
Partners should build an integration using [Run Task APIs in Terraform Cloud](https://www.terraform.io/docs/cloud/api/run-tasks.html). To better understand how run Task enhances the workflow, see diagram listed below and check out our [announcement about Terraform run Task](https://www.hashicorp.com/blog/terraform-cloud-run-tasks-beta-now-available). [Snyk](https://docs.snyk.io/features/integrations/ci-cd-integrations/integrating-snyk-with-terraform-cloud), for example, created an integration to detect configuration anomalies in code while reducing risk to the infrastructure. For additional API resources, [click here](https://www.terraform.io/docs/cloud/api/index.html).
**Currently, pre-apply is the only integration phase available.** 

![RunTask Diagram](/assets/images/docs/runtask-diagram.png)

### 3. Review

Schedule time with your Partner Alliance manager to review your integration. Demonstration of the integration will include but not limited to enabling the integration on the partner’s platform and Terraform Cloud, understanding of the use case for the integration as well as seeing the integration demonstrated live.  Alternatively, partners could also reach out to [technologypartners@hashicorp.com](technologypartners@hashicorp.com) if for some reason they are unable to engage with their Partner Alliances manager.

### 4. Release

Once demonstration has been completed and the documentation has been shared and verified. Partners will be added to the [Terraform Run Task page](https://www.terraform.io/docs/cloud/integrations/run-tasks/index.html#run-tasks-technology-partners).  On this page, partners will provide a two-line summary about their integration(s). If you have multiple integrations, we highly recommend creating a summary that highlights all potential integration options.  

Partners will provide documentation for end users to get started using their integration. [BridgeCrew](https://docs.bridgecrew.io/docs/integrate-with-terraform-cloud#introduction) embedded their documentation into their site and we also link to the documentation as well.   Also, partners will need to provide documentation for our support team including points of contact, email address, FAQ and/or best practices. With integrations, we want to ensure the end users are able to reach the right contacts for internal HashiCorp support when working with customers.

### 5. Support

Many vendors view the release step to be the end of the journey, while at HashiCorp we view it to be the beginning of the journey. Getting the integration built is just the first step in enabling users to leverage it against their infrastructure. Once development is completed, on-going effort is required to support the developed integration, maintain the integration and address any issues in a timely manner.

The expectation from the partner is to create a mechanism for them to track and resolve all critical issues as soon as possible within 48 hours and all other issues within 5 business days. This is a requirement given the critical nature of Terraform Cloud to customer’s operation. Vendors who choose to not support their integration will not be considered a verified integration and cannot be listed on the website.


-> Contact us at [technologypartners@hashicorp.com](technologypartners@hashicorp.com) with any questions, or feedback if you have other integration ideas. 
