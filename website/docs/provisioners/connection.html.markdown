---
layout: "docs"
page_title: "Provisioner Connections"
sidebar_current: "docs-provisioners-connection"
description: |-
  Managing connection defaults for SSH and WinRM using the `connection` block.
---

# Provisioner Connections

Many provisioners require access to the remote resource. For example,
a provisioner may need to use SSH or WinRM to connect to the resource.

Terraform uses a number of defaults when connecting to a resource, but these can
be overridden using a `connection` block in either a `resource` or
`provisioner`. Any `connection` information provided in a `resource` will apply
to all the provisioners, but it can be scoped to a single provisioner as well.
One use case is to have an initial provisioner connect as the `root` user to
setup user accounts, and have subsequent provisioners connect as a user with
more limited permissions.

## Example usage

```hcl
# Copies the file as the root user using SSH
provisioner "file" {
  source      = "conf/myapp.conf"
  destination = "/etc/myapp.conf"

  connection {
    type     = "ssh"
    user     = "root"
    password = "${var.root_password}"
  }
}

# Copies the file as the Administrator user using WinRM
provisioner "file" {
  source      = "conf/myapp.conf"
  destination = "C:/App/myapp.conf"

  connection {
    type     = "winrm"
    user     = "Administrator"
    password = "${var.admin_password}"
  }
}
```

## Argument Reference

**The following arguments are supported by all connection types:**

* `type` - The connection type that should be used. Valid types are `ssh` and `winrm`
  Defaults to `ssh`.

* `user` - The user that we should use for the connection. Defaults to `root` when
  using type `ssh` and defaults to `Administrator` when using type `winrm`.

* `password` - The password we should use for the connection. In some cases this is
  specified by the provider.

* `host` - The address of the resource to connect to. This is usually specified by the provider.

* `port` - The port to connect to. Defaults to `22` when using type `ssh` and defaults
  to `5985` when using type `winrm`.

* `timeout` - The timeout to wait for the connection to become available. This defaults
  to 5 minutes. Should be provided as a string like `30s` or `5m`.

* `script_path` - The path used to copy scripts meant for remote execution.

**Additional arguments only supported by the `ssh` connection type:**

* `private_key` - The contents of an SSH key to use for the connection. These can
  be loaded from a file on disk using the [`file()` interpolation
  function](/docs/configuration/interpolation.html#file_path_). This takes
  preference over the password if provided.

* `agent` - Set to `false` to disable using `ssh-agent` to authenticate. On Windows the
  only supported SSH authentication agent is
  [Pageant](http://the.earth.li/~sgtatham/putty/0.66/htmldoc/Chapter9.html#pageant).

**Additional arguments only supported by the `winrm` connection type:**

* `https` - Set to `true` to connect using HTTPS instead of HTTP.

* `insecure` - Set to `true` to not validate the HTTPS certificate chain.

* `cacert` - The CA certificate to validate against.

<a id="bastion"></a>
## Connecting through a Bastion Host with SSH

The `ssh` connection also supports the following fields to facilitate connnections via a
[bastion host](https://en.wikipedia.org/wiki/Bastion_host).

* `bastion_host` - Setting this enables the bastion Host connection. This host
  will be connected to first, and then the `host` connection will be made from there.

* `bastion_port` - The port to use connect to the bastion host. Defaults to the
  value of the `port` field.

* `bastion_user` - The user for the connection to the bastion host. Defaults to
  the value of the `user` field.

* `bastion_password` - The password we should use for the bastion host.
  Defaults to the value of the `password` field.

* `bastion_private_key` - The contents of an SSH key file to use for the bastion
  host. These can be loaded from a file on disk using the [`file()`
  interpolation function](/docs/configuration/interpolation.html#file_path_).
  Defaults to the value of the `private_key` field.

### Using more than one Bastion Host with SSH

All of the fields described above can be used as a comma separated list. Except for the bastion_port which needs to remain a single value.

Missing values are filled with the default values accordingly.

#### Example usage

```hcl
# Using two Bastion hosts
provisioner "remote-exec" {
  inline = [
    "echo Connected through two Bastion hosts"
  ]

  connection {
    user             = "root"
    password         = "${var.root_password}"
    bastion_host     = "bastionhost1,bastionhost2"
    # bastionhost1 will be connected using user 'wheel' and 'wheelsecret'
    # bastionhost2 will default to 'root' and '${var.root_password}'
    bastion_user     = "wheel"
    bastion_password = "wheelsecret"
  }
}
```

### Using a transparent Bastion Host with SSH

Similar to the list of Bastion Hosts introduced above you can use environment variables to add additional Bastion Hosts.

The Bastion Hosts added in this way will be used first for all `ssh` connections. Inline Bastion Host definitions will be chained to the global ones accordingly.

These variable names can be used in accordance with the inline fields:

* `TRANSPARENT_BASTIONHOST` - Setting this enables the use of the transparent
  bastion hosts.

* `TRANSPARENT_BASTIONPORT` - The port to use connect to the transparent bastion   host. In this case a comma separated list can be used as opposed to the
  `bastion_port` inline field.

* `TRANSPARENT_BASTIONUSER` - The user name to use connect to the transparent
  bastion host.

* `TRANSPARENT_BASTIONPASSWORD` - The password to use connect to the transparent
  bastion host.

* `TRANSPARENT_BASTIONPRIVATEKEY` - The private key to use connect to the
  transparent bastion host. You cannot use Terraform's interpolation syntax
  inside the environment variables, therefore you have to put the key here.
  For example you can use this this bash syntax:
  _TRANSPARENT_BASTIONPRIVATEKEY=$(< ~/.ssh/keyfile)_

The default values are assigned as described above. Every missing value is filled with the corresponding field inside the `connection` block of your `ssh` resource.
