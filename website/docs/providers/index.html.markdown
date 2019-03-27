---
layout: "docs"
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
resources. Providers generally are an IaaS (e.g. AWS, GCP, Microsoft Azure,
OpenStack), PaaS (e.g. Heroku), or SaaS services (e.g. Terraform Enterprise,
DNSimple, CloudFlare).

Use the navigation to the left to find available providers by type or scroll
down to see all providers.

<div style="column-width: 14em;">


- [ACME](/docs/providers/acme/index.html)
- [Alicloud](/docs/providers/alicloud/index.html)
- [Archive](/docs/providers/archive/index.html)
- [Arukas](/docs/providers/arukas/index.html)
- [AWS](/docs/providers/aws/index.html)
- [Azure](/docs/providers/azurerm/index.html)
- [Azure Active Directory](/docs/providers/azuread/index.html)
- [Azure Stack](/docs/providers/azurestack/index.html)
- [Bitbucket](/docs/providers/bitbucket/index.html)
- [Brightbox](/docs/providers/brightbox/index.html)
- [CenturyLinkCloud](/docs/providers/clc/index.html)
- [Chef](/docs/providers/chef/index.html)
- [Circonus](/docs/providers/circonus/index.html)
- [Cisco ASA](/docs/providers/ciscoasa/index.html)
- [Cloudflare](/docs/providers/cloudflare/index.html)
- [CloudScale.ch](/docs/providers/cloudscale/index.html)
- [CloudStack](/docs/providers/cloudstack/index.html)
- [Cobbler](/docs/providers/cobbler/index.html)
- [Consul](/docs/providers/consul/index.html)
- [Datadog](/docs/providers/datadog/index.html)
- [DigitalOcean](/docs/providers/do/index.html)
- [DNS](/docs/providers/dns/index.html)
- [DNSimple](/docs/providers/dnsimple/index.html)
- [DNSMadeEasy](/docs/providers/dme/index.html)
- [Docker](/docs/providers/docker/index.html)
- [Dyn](/docs/providers/dyn/index.html)
- [External](/docs/providers/external/index.html)
- [F5 BIG-IP](/docs/providers/bigip/index.html)
- [Fastly](/docs/providers/fastly/index.html)
- [FlexibleEngine](/docs/providers/flexibleengine/index.html)
- [GitHub](/docs/providers/github/index.html)
- [Gitlab](/docs/providers/gitlab/index.html)
- [Google Cloud Platform](/docs/providers/google/index.html)
- [Grafana](/docs/providers/grafana/index.html)
- [Hedvig](/docs/providers/hedvig/index.html)
- [Helm](/docs/providers/helm/index.html)
- [Heroku](/docs/providers/heroku/index.html)
- [Hetzner Cloud](/docs/providers/hcloud/index.html)
- [HTTP](/docs/providers/http/index.html)
- [HuaweiCloud](/docs/providers/huaweicloud/index.html)
- [Icinga2](/docs/providers/icinga2/index.html)
- [Ignition](/docs/providers/ignition/index.html)
- [InfluxDB](/docs/providers/influxdb/index.html)
- [JDcloud](/docs/providers/jdcloud/index.html)
- [Kubernetes](/docs/providers/kubernetes/index.html)
- [Librato](/docs/providers/librato/index.html)
- [Linode](/docs/providers/linode/index.html)
- [Local](/docs/providers/local/index.html)
- [Logentries](/docs/providers/logentries/index.html)
- [LogicMonitor](/docs/providers/logicmonitor/index.html)
- [Mailgun](/docs/providers/mailgun/index.html)
- [MySQL](/docs/providers/mysql/index.html)
- [Naver Cloud](/docs/providers/ncloud/index.html)
- [Netlify](/docs/providers/netlify/index.html)
- [New Relic](/docs/providers/newrelic/index.html)
- [Nomad](/docs/providers/nomad/index.html)
- [NS1](/docs/providers/ns1/index.html)
- [Null](/docs/providers/null/index.html)
- [Nutanix](/docs/providers/nutanix/index.html)
- [1&1](/docs/providers/oneandone/index.html)
- [OpenStack](/docs/providers/openstack/index.html)
- [OpenTelekomCloud](/docs/providers/opentelekomcloud/index.html)
- [OpsGenie](/docs/providers/opsgenie/index.html)
- [Oracle Cloud Infrastructure](/docs/providers/oci/index.html)
- [Oracle Cloud Platform](/docs/providers/oraclepaas/index.html)
- [Oracle Public Cloud](/docs/providers/opc/index.html)
- [OVH](/docs/providers/ovh/index.html)
- [Packet](/docs/providers/packet/index.html)
- [PagerDuty](/docs/providers/pagerduty/index.html)
- [Palo Alto Networks](/docs/providers/panos/index.html)
- [PostgreSQL](/docs/providers/postgresql/index.html)
- [PowerDNS](/docs/providers/powerdns/index.html)
- [ProfitBricks](/docs/providers/profitbricks/index.html)
- [RabbitMQ](/docs/providers/rabbitmq/index.html)
- [Rancher](/docs/providers/rancher/index.html)
- [Random](/docs/providers/random/index.html)
- [RightScale](/docs/providers/rightscale/index.html)
- [Rundeck](/docs/providers/rundeck/index.html)
- [RunScope](/docs/providers/runscope/index.html)
- [Scaleway](/docs/providers/scaleway/index.html)
- [Selectel](/docs/providers/selectel/index.html)
- [Skytap](/docs/providers/skytap/index.html)
- [SoftLayer](/docs/providers/softlayer/index.html)
- [Spotinst](/docs/providers/spotinst/index.html)
- [StatusCake](/docs/providers/statuscake/index.html)
- [TelefonicaOpenCloud](/docs/providers/telefonicaopencloud/index.html)
- [Template](/docs/providers/template/index.html)
- [TencentCloud](/docs/providers/tencentcloud/index.html)
- [Terraform](/docs/providers/terraform/index.html)
- [Terraform Enterprise](/docs/providers/tfe/index.html)
- [TLS](/docs/providers/tls/index.html)
- [Triton](/docs/providers/triton/index.html)
- [UCloud](/docs/providers/ucloud/index.html)
- [UltraDNS](/docs/providers/ultradns/index.html)
- [Vault](/docs/providers/vault/index.html)
- [VMware NSX-T](/docs/providers/nsxt/index.html)
- [VMware vCloud Director](/docs/providers/vcd/index.html)
- [VMware vSphere](/docs/providers/vsphere/index.html)
- [Yandex](/docs/providers/yandex/index.html)


</div>

-----

More providers can be found on our [Community Providers](/docs/providers/type/community-index.html) page.
