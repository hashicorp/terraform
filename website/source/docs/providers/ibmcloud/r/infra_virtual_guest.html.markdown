---
layout: "ibmcloud"
page_title: "IBM Cloud: infra_virtual_guest"
sidebar_current: "docs-ibmcloud-resource-infra-virtual-guest"
description: |-
  Manages IBM Cloud infrastructure Virtual Guests.
---

# ibmcloud\_infra_virtual_guest

Provides virtual guest resource. This allows virtual guests to be created, updated
and deleted. For additional details please refer to [API documentation](http://sldn.softlayer.com/reference/services/SoftLayer_Virtual_Guest).

## Example Usage

Create a new virtual guest using the "Debian" image.

```hcl
resource "ibmcloud_infra_virtual_guest" "twc_terraform_sample" {
    hostname = "twc-terraform-sample-name"
    domain = "bar.example.com"
    os_reference_code = "DEBIAN_7_64"
    datacenter = "wdc01"
    network_speed = 10
    hourly_billing = true
    private_network_only = false
    cores = 1
    memory = 1024
    disks = [25, 10, 20]
    user_metadata = "{\"value\":\"newvalue\"}"
    dedicated_acct_host_only = true
    local_disk = false
    public_vlan_id = 1391277
    private_vlan_id = 7721931
}
```

Create a new virtual guest using block device template.

```hcl
resource "ibmcloud_infra_virtual_guest" "terraform-sample-BDTGroup" {
   hostname = "terraform-sample-blockDeviceTemplateGroup"
   domain = "bar.example.com"
   datacenter = "ams01"
   public_network_speed = 10
   hourly_billing = false
   cores = 1
   memory = 1024
   local_disk = false
   image_id = 12345
   tags = [
     "collectd",
     "mesos-master"
   ]
   public_subnet = "50.97.46.160/28"
   private_subnet = "10.56.109.128/26"
}
```

## Argument Reference

The following arguments are supported:

*   `hostname` - (Optional) Hostname for the computing instance.
*   `domain` - (Required)  Domain for the computing instance.
*   `cores` - (Required) The number of CPU cores to allocate.
*   `memory` - (Required) The amount of memory to allocate in megabytes.
*   `datacenter` -  (Required) Specifies which datacenter the instance is to be provisioned in.
*   `hourly_billing` - (Optional, Default:true)Specifies the billing type for the instance. When true the computing instance will be billed on hourly usage, otherwise it will be billed on a monthly basis.
*   `local_disk`- (Optional, Default:true) Specifies the disk type for the instance. When true the disks for the computing instance will be provisioned on the host which it runs, otherwise SAN disks will be provisioned.
*   `dedicated_acct_host_only` - (Optional, Default:false) Specifies whether or not the instance must only run on hosts with instances from the same account
*   `os_reference_code` - (Optional) An operating system reference code that will be used to provision the computing instance. [Get a complete list of the os reference codes available](https://api.softlayer.com/rest/v3/SoftLayer_Virtual_Guest_Block_Device_Template_Group/getVhdImportSoftwareDescriptions.json?objectMask=referenceCode) (use your api key as the password). Conflicts with`image_id`.
*   `image_id` - (Optional) The image template id to be used to provision the computing instance. Note this is not the global identifier (uuid), but the image template group id that should point to a valid global identifier. You can get the image template id by navigating on the portal to _Devices > Manage > Images_, clicking on the desired image, and taking note of the id number in the browser URL location. Conflicts with `os_reference_code`.
**Note:** Don't know the ID(s) for your image templates? [You can reference them by name, too](https://github.com/softlayer/terraform-provider-softlayer/blob/master/docs/datasources/softlayer_image_template.md).
*   `network_speed` - (Optional, Default:100) Specifies the connection speed (in Mbps) for the instance's network components.
*   `private_network_only` - (Optional, Default:false) Specifies whether or not the instance only has access to the private network. When true this flag specifies that a compute instance is to only have access to the private network.
*   `public_vlan_id` - (Optional) Public VLAN id which is to be used for the public network interface of the instance. Accepted values can be found [here](https://control.softlayer.com/network/vlans). Click on the desired VLAN and note the ID on the resulting URL. Or, you can also [refer to a VLAN by name using a data source](https://github.com/softlayer/terraform-provider-softlayer/blob/master/docs/datasources/softlayer_vlan.md).
*   `private_vlan_id` - (Optional) Private VLAN id which is to be used for the private network interface of the instance. Accepted values can be found [here](https://control.softlayer.com/network/vlans). Click on the desired VLAN and note the ID on the resulting URL. Or, you can also [refer to a VLAN by name using a data source](https://github.com/softlayer/terraform-provider-softlayer/blob/master/docs/datasources/softlayer_vlan.md).

*   `public_subnet` - (Optional) Public subnet which is to be used for the public network interface of the instance. Accepted values are primary public networks and can be found [here](https://control.softlayer.com/network/subnets).
*   `private_subnet` - (Optional) Private subnet which is to be used for the private network interface of the instance. Accepted values are primary private networks and can be found [here](https://control.softlayer.com/network/subnets).
*   `disks` - (Optional) An *array* of numeric disk sizes (in GBs).Block device and disk image settings for the computing instance. Defaults to the smallest available capacity for the primary disk will be used. If an image template is specified the disk capacity will be be provided by the template.
*   `user_metadata` - (Optional) Arbitrary data to be made available to the computing instance.
*   `ssh_key_ids` - (Optional) An *array* of numbers. SSH key ids to install on the computing instance upon provisioning.
    **Note:** Don't know the ID(s) for your SSH keys? See [here](https://github.com/softlayer/terraform-provider-softlayer/blob/master/docs/datasources/softlayer_ssh_key.md) for a way to reference your SSH keys by their labels.

*   `post_install_script_uri` - (Optional)  As defined in the [SoftLayer_Virtual_Guest_SupplementalCreateObjectOptions](https://sldn.softlayer.com/reference/datatypes/SoftLayer_Virtual_Guest_SupplementalCreateObjectOptions).
*   `tags` - (Optional) An *array* of strings. Set tags on this virtual guest. The characters permitted are A-Z, 0-9, whitespace, _ (underscore), - (hyphen), . (period), and : (colon). All other characters will be stripped away.
*   `ipv6_enabled` - (Optional, Default:false) Provides a primary public IPv6 address.
*   `secondary_ip_count` - (Optional) Provides secondary public IPv4 addresses. Acceptable values are 4 and 8. 
*   `wait_time_minutes` - (Optional, Default:90) Wait this many minutes for the virtual guest to become available before declaring it as created. It is also the same amount of time waited for no active transactions before proceeding with an update or deletion.


## Attributes Reference

The following attributes are exported:

* `id` - id of the virtual guest.
* `ipv4_address` - Public IPv4 address of the virtual guest.
* `ip_address_id_private` - Unique ID for the private IPv4 address assigned to the virtual_guest.
* `ipv4_address_private` - Private IPv4 address of the virtual guest.
* `ip_address_id` - Unique ID for the public IPv4 address assigned to the virtual_guest.
* `ipv6_address` - Public IPv6 address of the virtual guest. It is provided when `ipv6_enabled` is `true`.
* `ipv6_address_id` - Unique ID for the public IPv6 address assigned to the virtual_guest. It is provided when `ipv6_enabled` is `true`.
* `public_ipv6_subnet` - Public IPv6 subnet. It is provided when `ipv6_enabled` is `true`.
* `secondary_ip_addresses` - Public secondary IPv4 addresses of the virtual guest.