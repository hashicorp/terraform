---
layout: "docs"
page_title: "Provisioner Connections"
sidebar_current: "docs-provisioners-connection"
---

# Provisioner Connections

Many provisioners require access to the remote resource. For example,
a provisioner may need to use ssh to connect to the resource.

Terraform uses a number of defaults when connecting to a resource, but these
can be overridden using `connection` block in either a `resource` or `provisioner`.
Any `connection` information provided in a `resource` will apply to all the
provisioners, but it can be scoped to a single provisioner as well. One use case
is to have an initial provisioner connect as root to setup user accounts, and have
subsequent provisioners connect as a user with more limited permissions.

## Example usage

```
# Copies the file as the root user using a password
provisioner "file" {
    source = "conf/myapp.conf"
    destination = "/etc/myapp.conf"
    connection {
        user = "root"
        password = "${var.root_password}"
    }
}
```

## Argument Reference

The following arguments are supported:

* `type` - The connection type that should be used. This defaults to "ssh". The type
  of connection supported depends on the provisioner.

* `user` - The user that we should use for the connection. This defaults to "root".

* `password` - The password we should use for the connection.

* `key_file` - The SSH key to use for the connection. This takes preference over the
   password if provided.

* `host` - The address of the resource to connect to. This is provided by the provider.

* `port` - The port to connect to. This defaults to 22.

* `timeout` - The timeout to wait for the connection to become available. This defaults
   to 5 minutes. Should be provided as a string like "30s" or "5m".

