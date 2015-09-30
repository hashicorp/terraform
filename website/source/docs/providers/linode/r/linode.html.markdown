---
layout: "linode"
page_title: "Linode: linode_linode"
description: |-
  Provides a Linode linode resource. This can be used to create, modify, and delete linodes.
---
# linode\_linode

Provides a Linode linode resource. This can be used to create, modify, and delete linodes.

## Example Usage

```
resource "linode_linode" "web" {
    image = "Ubuntu 14.04 LTS"
    kernel = "Latest 64 bit"
    name = "web"
    group = "integration"
    region = "Dallas, TX, USA"
    size = 1024
    private_networking = true
    ssh_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCxtdizvJzTT38y2oXuoLUXbLUf9V0Jy9KsM0bgIvjUCSEbuLWCXKnWqgBmkv7iTKGZg3fx6JA10hiufdGHD7at5YaRUitGP2mvC2I68AYNZmLCGXh0hYMrrUB01OEXHaYhpSmXIBc9zUdTreL5CvYe3PAYzuBA0/lGFTnNsHosSd+suA4xfJWMr/Fr4/uxrpcy8N8BE16pm4kci5tcMh6rGUGtDEj6aE9k8OI4SRmSZJsNElsu/Z/K4zqCpkW/U06vOnRrE98j3NE07nxVOTqdAMZqopFiMP0MXWvd6XyS2/uKU+COLLc0+hVsgj+dVMTWfy8wZ58OJDsIKk/cI/7yF+GZz89Js+qYx7u9mNhpEgD4UrcRHpitlRgVhA8p6R4oBqb0m/rpKBd2BAFdcty3GIP9CWsARtsCbN6YDLJ1JN3xI34jSGC1ROktVHg27bEEiT5A75w3WJl96BlSo5zJsIZDTWlaqnr26YxNHba4ILdVLKigQtQpf8WFsnB9YzmDdb9K3w9szf5lAkb/SFXw+e+yPS9habkpOncL0oCsgag5wUGCEmZ7wpiY8QgARhuwsQUkxv1aUi/Nn7b7sAkKSkxtBI3LBXZ+vcUxZTH0ut4pe9rbrEed3ktAOF5FafjA1VtarPqqZ+g46xVO9llgpXcl3rVglFtXzTcUy09hGw== btobolaski@Brendans-MacBook-Pro.local"
    root_password = "terraform-test"
}
```

## Argument Reference

The following arguments are supported:

* `image` - (Required) The image to use to build the linode. Can be either the
  name of one of the linode provider images or a custom image. Note that linode
  images are selected in preference to your custom images.
* `kernel` - (Required) The kernel version to use. It must be one of the
  kernels that Linode provides. You can also use `Latest 64 bit` to use
  Linode's most recent 64 bit kernel.
* `name` - (Optional) The name that is displayed in Linode's web ui
* `group` - (Optional) The group that the linode is diplayed under in the web ui
* `region` - (Required) The Linode region to create the server in. The value
  must match the [Location string here][1] exactly.
* `size` - (Required) The number of megabytes of ram in the size that you would
  like.
* `private_networking` - (Optional) Whether or not to enable private
  networking. **NOTE** This can only be enabled on an active server. Once it
  is enabled, it can't be disabled without destroying the server.
* `ssh_key` - (Required) A public key to add to the root user's authorized keys
  file. *Note* that this is ignored for custom images.
* `root_password` - (Required) The root password for the server that you
  created. Unfortunately, Linode's api requires this. Be sure that your
  provisioning processes removes the password. *Note* that this is ignored on
  custom images.
* `helper_distro` - (Optional) Enable the Distro filesystem helper. Corrects
  fstab and inittab/upstart entries depending on the kernel you're booting.
* `manage_private_ip_automatically` - (Optional) Whether or not to
  automatically set up the private ip address. If this is not enabled, you'll
  need to manually configure the server to have the specified private ip
  address.

[1]:https://www.linode.com/api/utility/avail.datacenters

## Attribute Reference

The following attributes are exported:

* `image` - The image that this linode is using
* `kernel` - The version of the kernel that this linode is using
* `name` - The display name of this linode
* `group` - The display group of this linode
* `region` - The region that this linode is in
* `size` - The size identifier for this linode
* `status` - a numerical value indicating this linode's status. Status values
  are -1: Being Created, 0: Brand New, 1: Running, and 2: Powered Off.
* `ip_address` - The linode's public ip address
* `private_networking` - Whether private networking is enabled for this linode.
* `private_ip_address` - If private networking is enabled, this is the private
  ip of the linode. Depending on the setting for
  `manage_private_ip_automatically`, this may need to be manually configured on
  the linode for it to be usable.
* `helper_distro` - Whether the distro helper is enabled.
* `manage_private_ip_automatically` - wether the private ip is handled
  automatically
