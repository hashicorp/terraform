---
layout: "docs"
page_title: "Provisioner: remote-exec"
sidebar_current: "docs-provisioners-remote"
---

# remote-exec Provisioner

The `remote-exec` provisioner invokes a script on a remote resource after it
is created. This can be used to run a configuration management tool, bootstrap
into a cluster, etc. To invoke a local process, see the `local-exec`
[provisioner](/docs/provisioners/local-exec.html) instead. The `remote-exec`
provisioner only supports `ssh` type [connections](/docs/provisioners/connection.html).


## Example usage

```
# Run puppet and join our Consul cluster
resource "aws_instance" "web" {
    ...
    provisioner "remote-exec" {
        inline = [
        "puppet apply",
        "consul join ${aws_instance.web.private_ip}",
        ]
    }
}
```

## Argument Reference

The following arguments are supported:

* `inline` - This is a list of command strings. They are executed in the order
  they are provided. This cannot be provided with `script` or `scripts`.

* `script` - This is a path (relative or absolute) to a local script that will
  be copied to the remote resource and then executed. This cannot be provided
  with `inline` or `scripts`.

* `scripts` - This is a list of paths (relative or absolute) to local scripts
  that will be copied to the remote resource and then executed. They are executed
  in the order they are provided. This cannot be provided with `inline` or `script`.

