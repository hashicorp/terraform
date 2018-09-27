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

<table class="table">

    <td><a href="/docs/providers/acme/index.html">ACME</a></td>
    <td><a href="/docs/providers/alicloud/index.html">Alicloud</a></td>
    <td><a href="/docs/providers/archive/index.html">Archive</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/arukas/index.html">Arukas</a></td>
    <td><a href="/docs/providers/aws/index.html">AWS</a></td>
    <td><a href="/docs/providers/azurerm/index.html">Azure</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/azurestack/index.html">Azure Stack</a></td>
    <td><a href="/docs/providers/bitbucket/index.html">Bitbucket</a></td>
    <td><a href="/docs/providers/brightbox/index.html">Brightbox</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/clc/index.html">CenturyLinkCloud</a></td>
    <td><a href="/docs/providers/chef/index.html">Chef</a></td>
    <td><a href="/docs/providers/circonus/index.html">Circonus</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/cloudflare/index.html">Cloudflare</a></td>
    <td><a href="/docs/providers/cloudscale/index.html">CloudScale.ch</a></td>
    <td><a href="/docs/providers/cloudstack/index.html">CloudStack</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/cobbler/index.html">Cobbler</a></td>
    <td><a href="/docs/providers/consul/index.html">Consul</a></td>
    <td><a href="/docs/providers/datadog/index.html">Datadog</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/do/index.html">DigitalOcean</a></td>
    <td><a href="/docs/providers/dns/index.html">DNS</a></td>
    <td><a href="/docs/providers/dme/index.html">DNSMadeEasy</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/dnsimple/index.html">DNSimple</a></td>
    <td><a href="/docs/providers/docker/index.html">Docker</a></td>
    <td><a href="/docs/providers/dyn/index.html">Dyn</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/external/index.html">External</a></td>
    <td><a href="/docs/providers/bigip/index.html">F5 BIG-IP</a></td>
    <td><a href="/docs/providers/fastly/index.html">Fastly</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/flexibleengine/index.html">FlexibleEngine</a></td>
    <td><a href="/docs/providers/github/index.html">GitHub</a></td>
    <td><a href="/docs/providers/gitlab/index.html">Gitlab</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/google/index.html">Google Cloud</a></td>
    <td><a href="/docs/providers/grafana/index.html">Grafana</a></td>
    <td><a href="/docs/providers/heroku/index.html">Heroku</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/hcloud/index.html">Hetzner Cloud</a></td>
    <td><a href="/docs/providers/http/index.html">HTTP</a></td>
    <td><a href="/docs/providers/huaweicloud/index.html">HuaweiCloud</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/icinga2/index.html">Icinga2</a></td>
    <td><a href="/docs/providers/ignition/index.html">Ignition</a></td>
    <td><a href="/docs/providers/influxdb/index.html">InfluxDB</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/kubernetes/index.html">Kubernetes</a></td>
    <td><a href="/docs/providers/librato/index.html">Librato</a></td>
    <td><a href="/docs/providers/local/index.html">Local</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/logentries/index.html">Logentries</a></td>
    <td><a href="/docs/providers/logicmonitor/index.html">LogicMonitor</a></td>
    <td><a href="/docs/providers/mailgun/index.html">Mailgun</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/mysql/index.html">MySQL</a></td>
    <td><a href="/docs/providers/netlify/index.html">Netlify</a></td>
    <td><a href="/docs/providers/newrelic/index.html">New Relic</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/nomad/index.html">Nomad</a></td>
    <td><a href="/docs/providers/ns1/index.html">NS1</a></td>
    <td><a href="/docs/providers/null/index.html">Null</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/oneandone/index.html">1&1</a></td>
    <td><a href="/docs/providers/openstack/index.html">OpenStack</a></td>
    <td><a href="/docs/providers/opentelekomcloud/index.html">OpenTelekomCloud</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/opsgenie/index.html">OpsGenie</a></td>
    <td><a href="/docs/providers/oci/index.html">Oracle Cloud Infrastructure</a></td>
    <td><a href="/docs/providers/oraclepaas/index.html">Oracle Cloud Platform</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/opc/index.html">Oracle Public Cloud</a></td>
    <td><a href="/docs/providers/ovh/index.html">OVH</a></td>
    <td><a href="/docs/providers/packet/index.html">Packet</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/pagerduty/index.html">PagerDuty</a></td>
    <td><a href="/docs/providers/panos/index.html">Palo Alto Networks</a></td>
    <td><a href="/docs/providers/postgresql/index.html">PostgreSQL</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/powerdns/index.html">PowerDNS</a></td>
    <td><a href="/docs/providers/profitbricks/index.html">ProfitBricks</a></td>
    <td><a href="/docs/providers/rabbitmq/index.html">RabbitMQ</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/rancher/index.html">Rancher</a></td>
    <td><a href="/docs/providers/random/index.html">Random</a></td>
    <td><a href="/docs/providers/rightscale/index.html">RightScale</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/rundeck/index.html">Rundeck</a></td>
    <td><a href="/docs/providers/runscope/index.html">RunScope</a></td>
    <td><a href="/docs/providers/scaleway/index.html">Scaleway</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/softlayer/index.html">SoftLayer</a></td>
    <td><a href="/docs/providers/statuscake/index.html">StatusCake</a></td>
    <td><a href="/docs/providers/spotinst/index.html">Spotinst</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/telefonicaopencloud/index.html">TelefonicaOpenCloud</a></td>
    <td><a href="/docs/providers/template/index.html">Template</a></td>
    <td><a href="/docs/providers/tencentcloud/index.html">TencentCloud</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/terraform/index.html">Terraform</a></td>
    <td><a href="/docs/providers/tfe/index.html">Terraform Enterprise</a></td>
    <td><a href="/docs/providers/tls/index.html">TLS</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/triton/index.html">Triton</a></td>
    <td><a href="/docs/providers/ultradns/index.html">UltraDNS</a></td>
    <td><a href="/docs/providers/vault/index.html">Vault</a></td>
    </tr>
    <tr>
    <td><a href="/docs/providers/vcd/index.html">VMware vCloud Director</a></td>
    <td><a href="/docs/providers/nsxt/index.html">VMware NSX-T</a></td>
    <td><a href="/docs/providers/vsphere/index.html">VMware vSphere</a></td>
    </tr>
</table>


More providers can be found on our [Community Providers](/docs/providers/type/community-index.html) page.
