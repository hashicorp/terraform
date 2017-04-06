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

Terraform uses a number of defaults when connecting to a resource, but these
can be overridden using a `connection` block in either a `resource` or `provisioner`.
Any `connection` information provided in a `resource` will apply to all the
provisioners, but it can be scoped to a single provisioner as well. One use case
is to have an initial provisioner connect as the `root` user to setup user accounts, and have
subsequent provisioners connect as a user with more limited permissions.

## Example usage

```
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

* `remote_forward` - The set of ports to forward from the remote host (the host
  being provisioned) to a given host on the local side of the SSH connection (the side
  running Terraform). The format is a comma separated list of
  forward instructions similar to `ssh -R`, i.e. `[bind_address]:port:host:hostport`.
  `bind_address` is the address to bind on the remote host. It is optional and defaults
  to `localhost`. `port` is the listen port on the remote side of the SSH connection.
  `host` is the host to connect to on the local side of the SSH connection.
  `hostport` is the port to connect to on the local side of the SSH connection.
  For example `localhost:8080:chef.example.com:80` will forward the port 8080 on the
  remote host (the host being provisioned) to port 80 of `chef.example.com` on the
  local side of the SSH connection (the host running Terraform). The feature is
  useful during bootstrapping if the host being provisioned
  does not yet have direct network access to required resources, e.g. a Chef Server.

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
