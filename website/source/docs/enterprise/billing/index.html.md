---
layout: "docs"
page_title: "Billing: Managed Nodes"
sidebar_current: "docs-enterprise"
description: |-
  HashiCorp charges for usage based on **managed nodes**. The definition of managed node is specific to the enterprise product and is described below.
---

# Managed Nodes

HashiCorp charges for usage based on **managed nodes**. The definition of
managed node is specific to the enterprise product and is described below.

For all enterprise products, the count of managed nodes is observed and
recorded every hour. At the end of the billing month a weighted average of
this recorded value is calculated to determine the overall managed node count
for billing.

## Terraform Enterprise

For Terraform Enterprise, a managed node is a compute resource defined in your
Terraform configuration. For certain resource types the managed node count is
determined by a property of the resource. The `count` meta-parameter is used
for all compute resource types. The complete list of compute resources and
resource arguments for determining managed node count is below.

| Provider | Resource Type | Resource Property |
|:-:|:-:|:-:|
| AWS | `aws_instance` | `count` |
| AWS | `aws_autoscaling_group` | `count` `desired_capacity` |
| Azure | `azure_instance` | `count` |
| Azure | `azurerm_virtual_machine` | `count` |
| CenturyLink Cloud | `clc_server` | `count` |
| CloudStack | `cloudstack_instance` | `count` |
| DigitalOcean | `digitalocean_droplet` | `count` |
| Google Cloud | `google_compute_instance` | `count` |
| Google Cloud | `compute_instance_group_manager` | `count` `target_size` |
| Heroku | `heroku_app` | `count` |
| OpenStack | `openstack_compute_instance_v2` | `count` |
| Packet | `packet_device` | `count` |
| Triton | `triton_machine` | `count` |
| VMware vCloud Director | `vcd_vapp` | `count` |
| VMware vSphere provider | `vsphere_virtual_machine` | `count` |


Terraform Enterprise includes unlimited Packer builds and artifact storage.

# Billing Support

For questions related to billing please email
[support@hashicorp.com](mailto:support@hashicorp.com).
