---
layout: "softlayer"
page_title: "SoftLayer: virtual_guest"
sidebar_current: "docs-softlayer-resource-virtual-guest"
description: |-
  Manages SoftLayer Virtual Guests.
---

# softlayer\virtual_guest

Provides virtual guest resource. This allows virtual guests to be created, updated
and deleted. For additional details please refer to [API documentation](http://sldn.softlayer.com/reference/services/SoftLayer_Virtual_Guest).

## Example Usage

```
# Create a new virtual guest using image "Debian"
resource "softlayer_virtual_guest" "twc_terraform_sample" {
    name = "twc-terraform-sample-name"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 10
    hourly_billing = true
    private_network_only = false
    cpu = 1
    ram = 1024
    disks = [25, 10, 20]
    user_data = "{\"value\":\"newvalue\"}"
    dedicated_acct_host_only = true
    local_disk = false
    frontend_vlan_id = 1085155
    backend_vlan_id = 1085157
}
```

```
# Create a new virtual guest using block device template
resource "softlayer_virtual_guest" "terraform-sample-BDTGroup" {
   name = "terraform-sample-blockDeviceTemplateGroup"
   domain = "bar.example.com"
   region = "ams01"
   public_network_speed = 10
   hourly_billing = false
   cpu = 1
   ram = 1024
   local_disk = false
   block_device_template_group_gid = "****-****-****-****-****"
}
```

## Argument Reference

The following arguments are supported:

* `name` | *string*
	* Hostname for the computing instance.
	* **Required**
* `domain` | *string*
	* Domain for the computing instance.
	* **Required**
* `cpu` | *int*
	* The number of CPU cores to allocate.
	* **Required**
* `ram` | *int*
	* The amount of memory to allocate in megabytes.
	* **Required**
* `region` | *string*
	* Specifies which datacenter the instance is to be provisioned in.
	* **Required**
* `hourly_billing` | *boolean*
	* Specifies the billing type for the instance. When true the computing instance will be billed on hourly usage, otherwise it will be billed on a monthly basis.
	* **Required**
* `local_disk` | *boolean*
	* Specifies the disk type for the instance. When true the disks for the computing instance will be provisioned on the host which it runs, otherwise SAN disks will be provisioned.
	* **Required**
* `dedicated_acct_host_only` | *boolean*
	* Specifies whether or not the instance must only run on hosts with instances from the same account
	* *Default*: nil
	* *Optional*
* `image` | *string*
	* An identifier for the operating system to provision the computing instance with.
	* **Conditionally required**	- Disallowed when blockDeviceTemplateGroup.globalIdentifier is provided, as the template will specify the operating system.
* `block_device_template_group_gid` | *string*
	* A global identifier for the template to be used to provision the computing instance.
	* **Conditionally required**	- Disallowed when operatingSystemReferenceCode is provided, as the template will specify the operating system.
* `public_network_speed` | *int*
	* Specifies the connection speed for the instance's network components.
	* *Default*: 10
	* *Optional*
* `private_network_only` | *boolean*
	* Specifies whether or not the instance only has access to the private network. When true this flag specifies that a compute instance is to only have access to the private network.
	* *Default*: False
	* *Optional*
* `frontend_vlan_id` | *int*
	* Specifies the network vlan which is to be used for the frontend interface of the computing instance.
	* *Default*: nil
	* *Optional*
* `backend_vlan_id` | *int*
	* Specifies the network vlan which is to be used for the backend interface of the computing instance.
	* *Default*: nil
	* *Optional*
* `disks` | *array*
	* Block device and disk image settings for the computing instance
	* *Optional*
	* *Default*: The smallest available capacity for the primary disk will be used. If an image template is specified the disk capacity will be be provided by the template.
* `user_data` | *string*
	* Arbitrary data to be made available to the computing instance.
	* *Default*: nil
	* *Optional*
* `ssh_keys` | *array*
	* SSH keys to install on the computing instance upon provisioning.
	* *Default*: nil
	* *Optional*
* `ipv4_address` | *string*
	* Uses editObject call, template data [defined here](https://sldn.softlayer.com/reference/datatypes/SoftLayer_Virtual_Guest).
	* *Default*: nil
	* *Optional*
* `ipv4_address_private` | *string*
	* Uses editObject call, template data [defined here](https://sldn.softlayer.com/reference/datatypes/SoftLayer_Virtual_Guest).
	* *Default*: nil
	* *Optional*
* `post_install_script_uri` | *string*
	* As defined in the [SoftLayer_Virtual_Guest_SupplementalCreateObjectOptions](https://sldn.softlayer.com/reference/datatypes/SoftLayer_Virtual_Guest_SupplementalCreateObjectOptions).
	* *Default*: nil
	* *Optional*
				
## Attributes Reference

The following attributes are exported:	

* `id` - id of the virtual guest.

	
