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

<table class="apidocs">
  <tr>
    <th>Provider</th>
    <th>Resource Type</th>
    <th>Resource Property</th>
  </tr>
  </tr>
  <tr>
    <td>AWS</td>
    <td>`aws_instance`</td>
    <td>`count`</td>
  </tr>
  <tr>    
    <td>AWS 
    <td>`aws_autoscaling_group`</td>
    <td>`count` `desired_capacity`</td>
  </tr>
  <tr>    
    <td>Azure</td> 
    <td>`azure_instance`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>Azure</td> 
    <td>`azurerm_virtual_machine`</td> 
    <td>`count`</td>
  </tr>
  <tr>    
    <td>CenturyLink Cloud</td> 
    <td>`clc_server`</td> 
    <td>`count`</td>
  </tr>
  <tr> 
    <td>CloudStack</td> 
    <td>`cloudstack_instance`</td> 
    <td>`count`</td>
  </tr>
  <tr> 
    <td>DigitalOcean</td> 
    <td>`digitalocean_droplet`</td> 
    <td>`count`</td>
  </tr>
  <tr>  
    <td>Google Cloud</td> 
    <td>`google_compute_instance`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>Google Cloud</td> 
    <td>`compute_instance_group_manager`</td> 
    <td>`count` `target_size`</td>
  </tr>
  <tr>  
    <td>Heroku</td> 
    <td>`heroku_app`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>OpenStack</td> 
    <td>`openstack_compute_instance_v2`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>Packet</td> 
    <td>`packet_device`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>Triton</td> 
    <td>`triton_machine`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>VMware vCloud Director</td> 
    <td>`vcd_vapp`</td> 
    <td>`count`</td>
  </tr>
  <tr>
    <td>VMware vSphere provider</td> 
    <td>`vsphere_virtual_machine`</td> 
    <td>`count`</td>
  </tr>
</table>

Terraform Enterprise includes unlimited Packer builds and artifact storage.

# Billing Support

For questions related to billing please email
[support@hashicorp.com](mailto:support@hashicorp.com).
