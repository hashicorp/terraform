---
layout: "azure"
page_title: "Azure: azure_instance"
sidebar_current: "docs-azure-resource-instance"
description: |-
  Creates a hosted service, role and deployment and then creates a virtual machine in the deployment based on the specified configuration.
---

# azure\_instance

Creates a hosted service, role and deployment and then creates a virtual
machine in the deployment based on the specified configuration.

## Example Usage

```
resource "azure_hosted_service" "terraform-service" {
    name = "terraform-service"
    location = "North Europe"
    ephemeral_contents = false
    description = "Hosted service created by Terraform."
    label = "tf-hs-01"
}

resource "azure_instance" "web" {
    name = "terraform-test"
    hosted_service_name = "${azure_hosted_service.terraform-service.name}"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "yourstorage"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"
    domain_name = "contoso.com"
    domain_ou = "OU=Servers,DC=contoso.com,DC=Contoso,DC=com"
    domain_username = "Administrator"
    domain_password = "Pa$$word123"

    endpoint {
        name = "SSH"
        protocol = "tcp"
        public_port = 22
        private_port = 22
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the instance. Changing this forces a new
    resource to be created.

* `hosted_service_name` - (Optional) The name of the hosted service the
    instance should be deployed under. If not provided; it will default to the
    value of `name`. Changes to this parameter forces the creation of a new
    resource.

* `description` - (Optional) The description for the associated hosted service.
    Changing this forces a new resource to be created (defaults to the instance
    name).

* `image` - (Required) The name of an existing VM or OS image to use for this
    instance. Changing this forces a new resource to be created.

* `size` - (Required) The size of the instance.

* `subnet` - (Optional) The name of the subnet to connect this instance to. If
    a value is supplied `virtual_network` is required. Changing this forces a
    new resource to be created.

* `virtual_network` - (Optional) The name of the virtual network the `subnet`
    belongs to. If a value is supplied `subnet` is required. Changing this
    forces a new resource to be created.

* `storage_service_name` - (Optional) The name of an existing storage account
    within the subscription which will be used to store the VHDs of this
    instance. Changing this forces a new resource to be created. **A Storage
    Service is required if you are using a Platform Image**

* `reverse_dns` - (Optional) The DNS address to which the IP address of the
    hosted service resolves when queried using a reverse DNS query. Changing
    this forces a new resource to be created.

* `location` - (Required) The location/region where the cloud service is
    created. Changing this forces a new resource to be created.

* `automatic_updates` - (Optional) If true this will enable automatic updates.
    This attribute is only used when creating a Windows instance. Changing this
    forces a new resource to be created (defaults false)

* `time_zone` - (Optional) The appropriate time zone for this instance in the
    format 'America/Los_Angeles'. This attribute is only used when creating a
    Windows instance. Changing this forces a new resource to be created
    (defaults false)

* `username` - (Required) The username of a new user that will be created while
    creating the instance. Changing this forces a new resource to be created.

* `password` - (Optional) The password of the new user that will be created
    while creating the instance. Required when creating a Windows instance or
    when not supplying an `ssh_key_thumbprint` while creating a Linux instance.
    Changing this forces a new resource to be created.

* `ssh_key_thumbprint` - (Optional) The SSH thumbprint of an existing SSH key
    within the subscription. This attribute is only used when creating a Linux
    instance. Changing this forces a new resource to be created.

* `security_group` - (Optional) The Network Security Group to associate with
    this instance.

* `endpoint` - (Optional) Can be specified multiple times to define multiple
    endpoints. Each `endpoint` block supports fields documented below.

* `domain_name` - (Optional) The name of an Active Directory domain to join.

* `domain_ou` - (Optional) Specifies the LDAP Organizational Unit to place the 
    instance in.

* `domain_username` - (Optional) The username of an account with permission to
    join the instance to the domain. Required if a domain_name is specified.

* `domain_password` - (Optional) The password for the domain_username account
    specified above.


The `endpoint` block supports:

* `name` - (Required) The name of the external endpoint.

* `protocol` - (Optional) The transport protocol for the endpoint. Valid
    options are: `tcp` and `udp` (defaults `tcp`)

* `public_port` - (Required) The external port to use for the endpoint.

* `private_port` - (Required) The private port on which the instance is
    listening.

## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.
* `description` - The description for the associated hosted service.
* `subnet` - The subnet the instance is connected to.
* `endpoint` - The complete set of configured endpoints.
* `security_group` - The associated Network Security Group.
* `ip_address` - The private IP address assigned to the instance.
* `vip_address` - The public IP address assigned to the instance.
