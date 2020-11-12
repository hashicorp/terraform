---
layout: "language"
page_title: "Providers"
sidebar_current: "docs-providers"
description: |-
  Terraform is used to create, manage, and manipulate infrastructure resources. Examples of resources include physical machines, VMs, network switches, containers, etc. Almost any infrastructure noun can be represented as a resource in Terraform.
---

# Providers

Terraform is used to create, manage, and update infrastructure resources such
as physical machines, VMs, network switches, containers, and more. Almost any
infrastructure type can be represented as a resource in Terraform.

A provider is responsible for understanding API interactions and exposing
resources. Most providers configure a specific infrastructure platform (either
cloud or self-hosted). Providers can also offer local utilities for tasks like
generating random numbers for unique resource names.

## Providers in the Terraform Registry

The [Terraform Registry](https://registry.terraform.io/browse/providers)
is the main directory of publicly available Terraform providers, and hosts
providers for most major infrastructure platforms.

Once you've found a provider you want to use, you can require it in your
Terraform configuration and start using the resource types it provides.
Terraform can automatically install providers from the Terraform Registry when
you run `terraform init`.

- To find providers for the infrastructure platforms you use, browse
  [the providers section of the Terraform Registry](https://registry.terraform.io/browse/providers).
- For details about how to use providers in your Terraform configurations, see
  [Provider Requirements](../configuration/provider-requirements.html) and
  [Provider Configuration](../configuration/providers.html).

### Provider Documentation

Every Terraform provider has its own documentation, describing its resource
types and their arguments.

The Terraform Registry is also the main home for provider documentation.
When viewing a provider's page on the Terraform Registry, you can click the
"Documentation" link in the header to browse its documentation. Provider
documentation in the registry is versioned, and you can use the dropdown version
menu in the header to switch which version's documentation you are viewing.

## Lists of Terraform Providers

Provider documentation used to be hosted directly on terraform.io, as part of
Terraform's core documentation. Although some provider documentation might still
be hosted here, the Terraform Registry is now the main home for all public
provider docs. (The exception is the built-in
[`terraform` provider](/docs/providers/terraform/index.html) for reading state
data, since it is not available on the Terraform Registry.)

As part of the old provider documentation, this section of the site included
categorized lists of all of the providers that could be automatically installed
by older versions of Terraform, plus a supplemental list of community providers
that needed to be manually installed. Many of these providers have already moved
to the Terraform Registry, but we will continue to host these lists for a while
as part of the transition. Links to provider documentation URLs on terraform.io
should still work, but will now redirect to the equivalent page in the Terraform
Registry.

Use the navigation to the left to browse the categorized lists, or see the main
list of historical providers below.

<div style="column-width: 14em;">


- [ACME](/docs/providers/acme/index.html)
- [Akamai](/docs/providers/akamai/index.html)
- [Alibaba Cloud](/docs/providers/alicloud/index.html)
- [Archive](/docs/providers/archive/index.html)
- [Arukas](/docs/providers/arukas/index.html)
- [Auth0](/docs/providers/auth0/index.html)
- [Avi Vantage](/docs/providers/avi/index.html)
- [Aviatrix](/docs/providers/aviatrix/index.html)
- [AWS](/docs/providers/aws/index.html)
- [Azure](/docs/providers/azurerm/index.html)
- [Azure Active Directory](/docs/providers/azuread/index.html)
- [Azure DevOps](/docs/providers/azuredevops/index.html)
- [Azure Stack](/docs/providers/azurestack/index.html)
- [A10 Networks](/docs/providers/vthunder/index.html)
- [BaiduCloud](/docs/providers/baiducloud/index.html)
- [Bitbucket](/docs/providers/bitbucket/index.html)
- [Brightbox](/docs/providers/brightbox/index.html)
- [CenturyLinkCloud](/docs/providers/clc/index.html)
- [Check Point](/docs/providers/checkpoint/index.html)
- [Chef](/docs/providers/chef/index.html)
- [CherryServers](/docs/providers/cherryservers/index.html)
- [Circonus](/docs/providers/circonus/index.html)
- [Cisco ASA](/docs/providers/ciscoasa/index.html)
- [Cisco ACI](/docs/providers/aci/index.html)
- [Cisco MSO](/docs/providers/mso/index.html)
- [CloudAMQP](/docs/providers/cloudamqp/index.html)
- [Cloudflare](/docs/providers/cloudflare/index.html)
- [Cloud-init](/docs/providers/cloudinit/index.html)
- [CloudScale.ch](/docs/providers/cloudscale/index.html)
- [CloudStack](/docs/providers/cloudstack/index.html)
- [Cobbler](/docs/providers/cobbler/index.html)
- [Cohesity](/docs/providers/cohesity/index.html)
- [Constellix](/docs/providers/constellix/index.html)
- [Consul](/docs/providers/consul/index.html)
- [Datadog](/docs/providers/datadog/index.html)
- [DigitalOcean](/docs/providers/do/index.html)
- [DNS](/docs/providers/dns/index.html)
- [DNSimple](/docs/providers/dnsimple/index.html)
- [DNSMadeEasy](/docs/providers/dme/index.html)
- [Docker](/docs/providers/docker/index.html)
- [Dome9](/docs/providers/dome9/index.html)
- [Dyn](/docs/providers/dyn/index.html)
- [EnterpriseCloud](/docs/providers/ecl/index.html)
- [Exoscale](/docs/providers/exoscale/index.html)
- [External](/docs/providers/external/index.html)
- [F5 BIG-IP](/docs/providers/bigip/index.html)
- [Fastly](/docs/providers/fastly/index.html)
- [FlexibleEngine](/docs/providers/flexibleengine/index.html)
- [FortiOS](/docs/providers/fortios/index.html)
- [Genymotion](/docs/providers/genymotion/index.html)
- [GitHub](/docs/providers/github/index.html)
- [GitLab](/docs/providers/gitlab/index.html)
- [Google Cloud Platform](/docs/providers/google/index.html)
- [Grafana](/docs/providers/grafana/index.html)
- [Gridscale](/docs/providers/gridscale)
- [Hedvig](/docs/providers/hedvig/index.html)
- [Helm](/docs/providers/helm/index.html)
- [Heroku](/docs/providers/heroku/index.html)
- [Hetzner Cloud](/docs/providers/hcloud/index.html)
- [HTTP](/docs/providers/http/index.html)
- [HuaweiCloud](/docs/providers/huaweicloud/index.html)
- [HuaweiCloudStack](/docs/providers/huaweicloudstack/index.html)
- [Icinga2](/docs/providers/icinga2/index.html)
- [Ignition](/docs/providers/ignition/index.html)
- [Incapsula](/docs/providers/incapsula/index.html)
- [InfluxDB](/docs/providers/influxdb/index.html)
- [Infoblox](/docs/providers/infoblox/index.html)
- [JDCloud](/docs/providers/jdcloud/index.html)
- [KingsoftCloud](/docs/providers/ksyun/index.html)
- [Kubernetes](/docs/providers/kubernetes/index.html)
- [Lacework](/docs/providers/lacework/index.html)
- [LaunchDarkly](/docs/providers/launchdarkly/index.html)
- [Librato](/docs/providers/librato/index.html)
- [Linode](/docs/providers/linode/index.html)
- [Local](/docs/providers/local/index.html)
- [Logentries](/docs/providers/logentries/index.html)
- [LogicMonitor](/docs/providers/logicmonitor/index.html)
- [Mailgun](/docs/providers/mailgun/index.html)
- [MetalCloud](/docs/providers/metalcloud/index.html)
- [MongoDB Atlas](/docs/providers/mongodbatlas/index.html)
- [MySQL](/docs/providers/mysql/index.html)
- [Naver Cloud](/docs/providers/ncloud/index.html)
- [Netlify](/docs/providers/netlify/index.html)
- [New Relic](https://registry.terraform.io/providers/newrelic/newrelic/latest/docs)
- [Nomad](/docs/providers/nomad/index.html)
- [NS1](/docs/providers/ns1/index.html)
- [Null](https://registry.terraform.io/providers/hashicorp/null/latest/docs)
- [Nutanix](/docs/providers/nutanix/index.html)
- [1&1](/docs/providers/oneandone/index.html)
- [Okta](/docs/providers/okta/index.html)
- [Okta Advanced Server Access](/docs/providers/oktaasa/index.html)
- [OpenNebula](/docs/providers/opennebula/index.html)
- [OpenStack](/docs/providers/openstack/index.html)
- [OpenTelekomCloud](/docs/providers/opentelekomcloud/index.html)
- [OpsGenie](/docs/providers/opsgenie/index.html)
- [Oracle Cloud Infrastructure](/docs/providers/oci/index.html)
- [Oracle Cloud Platform](/docs/providers/oraclepaas/index.html)
- [Oracle Public Cloud](/docs/providers/opc/index.html)
- [OVH](/docs/providers/ovh/index.html)
- [Packet](/docs/providers/packet/index.html)
- [PagerDuty](/docs/providers/pagerduty/index.html)
- [Palo Alto Networks PANOS](/docs/providers/panos/index.html)
- [Palo Alto Networks PrismaCloud](/docs/providers/prismacloud/index.html)
- [PostgreSQL](/docs/providers/postgresql/index.html)
- [PowerDNS](/docs/providers/powerdns/index.html)
- [ProfitBricks](/docs/providers/profitbricks/index.html)
- [Pureport](/docs/providers/pureport/index.html)
- [RabbitMQ](/docs/providers/rabbitmq/index.html)
- [Rancher](/docs/providers/rancher/index.html)
- [Rancher2](/docs/providers/rancher2/index.html)
- [Random](https://registry.terraform.io/providers/hashicorp/random/latest/docs)
- [RightScale](/docs/providers/rightscale/index.html)
- [Rubrik](/docs/providers/rubrik/index.html)
- [Rundeck](/docs/providers/rundeck/index.html)
- [RunScope](/docs/providers/runscope/index.html)
- [Scaleway](/docs/providers/scaleway/index.html)
- [Selectel](/docs/providers/selectel/index.html)
- [SignalFx](/docs/providers/signalfx/index.html)
- [Skytap](/docs/providers/skytap/index.html)
- [SoftLayer](/docs/providers/softlayer/index.html)
- [Spotinst](/docs/providers/spotinst/index.html)
- [StackPath](/docs/providers/stackpath/index.html)
- [StatusCake](/docs/providers/statuscake/index.html)
- [Sumo Logic](/docs/providers/sumologic/index.html)
- [TelefonicaOpenCloud](/docs/providers/telefonicaopencloud/index.html)
- [Template](/docs/providers/template/index.html)
- [TencentCloud](/docs/providers/tencentcloud/index.html)
- [Terraform](/docs/providers/terraform/index.html)
- [Terraform Cloud](/docs/providers/tfe/index.html)
- [Time](/docs/providers/time/index.html)
- [TLS](/docs/providers/tls/index.html)
- [Triton](/docs/providers/triton/index.html)
- [Turbot](/docs/providers/turbot/index.html)
- [UCloud](/docs/providers/ucloud/index.html)
- [UltraDNS](/docs/providers/ultradns/index.html)
- [Vault](/docs/providers/vault/index.html)
- [Venafi](/docs/providers/venafi/index.html)
- [VMware Cloud](/docs/providers/vmc/index.html)
- [VMware NSX-T](/docs/providers/nsxt/index.html)
- [VMware vCloud Director](/docs/providers/vcd/index.html)
- [VMware vRA7](/docs/providers/vra7/index.html)
- [VMware vSphere](/docs/providers/vsphere/index.html)
- [Vultr](/docs/providers/vultr/index.html)
- [Wavefront](/docs/providers/wavefront/index.html)
- [Yandex](/docs/providers/yandex/index.html)


</div>

-----

More providers can be found on our [Community Providers](/docs/providers/type/community-index.html) page.
